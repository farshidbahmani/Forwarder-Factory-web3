// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

interface IFactory {
    function motherWallet() external view returns (address);
    // FIX #1: added owner() to interface — avoids raw staticcall in onlyOwner modifier
    function owner() external view returns (address);
}

/// @title Forwarder
/// @notice Per-user wallet — a Minimal Proxy is cloned from this contract.
/// Protections:
///   - onlyFactory: only the Factory can trigger sweeps
///   - nonReentrant: prevents reentrancy attacks
///   - emergencyWithdraw: if the Factory fails, the Owner can rescue funds directly
///     Funds are always sent to motherWallet — no free `to` parameter
/// Known limitation:
///   - ERC-777 tokens trigger tokensReceived() hook on transfer;
///     nonReentrant guards against reentrancy, but avoid whitelisting ERC-777 tokens
///     in production without additional review.
contract Forwarder is ReentrancyGuard {
    using SafeERC20 for IERC20;

    address public immutable factory;

    event SweptToken(address indexed token, address indexed to, uint256 amount);
    event SweptNative(address indexed to, uint256 amount);
    event EmergencyWithdrawToken(address indexed token, address indexed to, uint256 amount);
    event EmergencyWithdrawNative(address indexed to, uint256 amount);
    // FIX #2: added Received event so every incoming ETH is traceable on-chain
    event Received(address indexed sender, uint256 amount);

    modifier onlyFactory() {
        require(msg.sender == factory, "Forwarder: not factory");
        _;
    }

    /// @dev Resolves the Factory owner via the typed IFactory interface.
    // FIX #1: replaced raw staticcall with IFactory.owner() — cleaner and safer
    modifier onlyOwner() {
        require(msg.sender == IFactory(factory).owner(), "Forwarder: not owner");
        _;
    }

    constructor(address _factory) {
        require(_factory != address(0), "Forwarder: zero factory");
        factory = _factory;
    }

    // ─────────────────────────────────────────
    // Normal sweep functions (Factory only)
    // ─────────────────────────────────────────

    function sweepToken(address token) external onlyFactory nonReentrant {
        // FIX #3: zero-address check on token
        require(token != address(0), "Forwarder: zero token");
        address mother = IFactory(factory).motherWallet();
        uint256 balance = IERC20(token).balanceOf(address(this));
        require(balance > 0, "Forwarder: zero token balance");
        IERC20(token).safeTransfer(mother, balance);
        emit SweptToken(token, mother, balance);
    }

    function sweepNative() external onlyFactory nonReentrant {
        address mother = IFactory(factory).motherWallet();
        uint256 balance = address(this).balance;
        require(balance > 0, "Forwarder: zero native balance");
        (bool success, ) = mother.call{value: balance}("");
        require(success, "Forwarder: native transfer failed");
        emit SweptNative(mother, balance);
    }

    // ─────────────────────────────────────────
    // Emergency functions (directly by Owner)
    // If the Factory has a bug or is unavailable,
    // the Owner can rescue funds directly.
    // FIX #4: funds always go to motherWallet — free `to` parameter removed
    //         to prevent drained funds in case of a compromised owner key
    // ─────────────────────────────────────────

    /// @notice Emergency BEP20 token withdrawal — Factory owner only.
    /// Funds are always sent to motherWallet (read from Factory at call time).
    function emergencyWithdrawToken(address token) external onlyOwner nonReentrant {
        // FIX #3: zero-address check on token
        require(token != address(0), "Forwarder: zero token");
        address mother = IFactory(factory).motherWallet();
        uint256 balance = IERC20(token).balanceOf(address(this));
        require(balance > 0, "Forwarder: zero token balance");
        IERC20(token).safeTransfer(mother, balance);
        emit EmergencyWithdrawToken(token, mother, balance);
    }

    /// @notice Emergency native token withdrawal — Factory owner only.
    /// Funds are always sent to motherWallet (read from Factory at call time).
    function emergencyWithdrawNative() external onlyOwner nonReentrant {
        address mother = IFactory(factory).motherWallet();
        uint256 balance = address(this).balance;
        require(balance > 0, "Forwarder: zero native balance");
        (bool success, ) = mother.call{value: balance}("");
        require(success, "Forwarder: native transfer failed");
        emit EmergencyWithdrawNative(mother, balance);
    }

    // FIX #2: emit Received event on every incoming ETH for on-chain traceability
    receive() external payable {
        emit Received(msg.sender, msg.value);
    }
}
