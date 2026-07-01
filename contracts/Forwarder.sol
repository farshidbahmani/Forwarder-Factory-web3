// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

interface IFactory {
    function motherWallet() external view returns (address);
}

/// @title Forwarder
/// @notice Per-user wallet — a Minimal Proxy is cloned from this contract.
/// Protections:
///   - onlyFactory: only the Factory can trigger sweeps
///   - nonReentrant: prevents reentrancy attacks
///   - emergencyWithdraw: if the Factory fails, the Owner can rescue funds directly
contract Forwarder is ReentrancyGuard {
    using SafeERC20 for IERC20;

    address public immutable factory;

    event SweptToken(address indexed token, address indexed to, uint256 amount);
    event SweptBNB(address indexed to, uint256 amount);
    event EmergencyWithdrawToken(address indexed token, address indexed to, uint256 amount);
    event EmergencyWithdrawBNB(address indexed to, uint256 amount);

    modifier onlyFactory() {
        require(msg.sender == factory, "Forwarder: not factory");
        _;
    }

    /// @dev Only the Factory owner can perform emergency withdrawals
    modifier onlyOwner() {
        (bool ok, bytes memory data) = factory.staticcall(
            abi.encodeWithSignature("owner()")
        );
        require(ok, "Forwarder: owner call failed");
        address factoryOwner = abi.decode(data, (address));
        require(msg.sender == factoryOwner, "Forwarder: not owner");
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
        address mother = IFactory(factory).motherWallet();
        uint256 balance = IERC20(token).balanceOf(address(this));
        require(balance > 0, "Forwarder: zero token balance");
        IERC20(token).safeTransfer(mother, balance);
        emit SweptToken(token, mother, balance);
    }

    function sweepBNB() external onlyFactory nonReentrant {
        address mother = IFactory(factory).motherWallet();
        uint256 balance = address(this).balance;
        require(balance > 0, "Forwarder: zero BNB balance");
        (bool success, ) = mother.call{value: balance}("");
        require(success, "Forwarder: BNB transfer failed");
        emit SweptBNB(mother, balance);
    }

    // ─────────────────────────────────────────
    // Emergency functions (directly by Owner)
    // If the Factory has a bug or is unavailable,
    // the Owner can rescue funds directly
    // ─────────────────────────────────────────

    /// @notice Emergency BEP20 token withdrawal — Factory owner only
    function emergencyWithdrawToken(
        address token,
        address to
    ) external onlyOwner nonReentrant {
        require(to != address(0), "Forwarder: zero to");
        uint256 balance = IERC20(token).balanceOf(address(this));
        require(balance > 0, "Forwarder: zero token balance");
        IERC20(token).safeTransfer(to, balance);
        emit EmergencyWithdrawToken(token, to, balance);
    }

    /// @notice Emergency BNB withdrawal — Factory owner only
    function emergencyWithdrawBNB(address to) external onlyOwner nonReentrant {
        require(to != address(0), "Forwarder: zero to");
        uint256 balance = address(this).balance;
        require(balance > 0, "Forwarder: zero BNB balance");
        (bool success, ) = to.call{value: balance}("");
        require(success, "Forwarder: BNB transfer failed");
        emit EmergencyWithdrawBNB(to, balance);
    }

    receive() external payable {}
}
