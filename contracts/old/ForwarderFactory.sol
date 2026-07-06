// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/proxy/Clones.sol";
import "./Forwarder.sol";

/// @title ForwarderFactory
/// @notice Additional protections:
///   - Timelock: motherWallet changes must be announced 48 hours in advance
///   - emergencyWithdraw: Owner can rescue funds from any user wallet directly
///   - Two-step ownership transfer: prevents accidental loss of ownership
///   - Events emitted for all important state changes
contract ForwarderFactory {
    address public immutable implementation;

    address public motherWallet;
    address public relayer;
    address public owner;

    // ─── Two-step ownership transfer ───
    // pendingOwner must call acceptOwnership() to confirm.
    // Prevents permanent loss of ownership due to a typo.
    address public pendingOwner;

    // ─── Timelock for Mother Wallet changes ───
    uint256 public constant TIMELOCK_DELAY = 48 hours;
    address public pendingMotherWallet;
    uint256 public motherWalletUnlockTime;

    event WalletDeployed(uint256 indexed userId, address wallet);
    event RelayerUpdated(address oldRelayer, address newRelayer);
    event MotherWalletChangeRequested(address newMotherWallet, uint256 unlockTime);
    event MotherWalletUpdated(address oldMotherWallet, address newMotherWallet);
    event MotherWalletChangeCancelled(address cancelledAddress);

    // Step 1: current owner nominates a new owner
    event OwnershipTransferStarted(address indexed currentOwner, address indexed pendingOwner);
    // Step 2: pending owner accepts — ownership is finalized
    event OwnershipTransferred(address indexed oldOwner, address indexed newOwner);
    // Optional: current owner cancels before acceptance
    event OwnershipTransferCancelled(address indexed cancelledPendingOwner);

    event EmergencyWithdraw(uint256 indexed userId, address token, address to);

    modifier onlyOwner() {
        require(msg.sender == owner, "Factory: not owner");
        _;
    }

    modifier onlyRelayer() {
        require(msg.sender == relayer, "Factory: not relayer");
        _;
    }

    constructor(address _motherWallet, address _relayer) {
        require(_motherWallet != address(0), "Factory: zero mother wallet");
        require(_relayer != address(0), "Factory: zero relayer");
        owner = msg.sender;
        motherWallet = _motherWallet;
        relayer = _relayer;
        implementation = address(new Forwarder(address(this)));
    }

    function _salt(uint256 userId) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(userId));
    }

    // ─────────────────────────────────────────
    // Core functions
    // ─────────────────────────────────────────

    function getAddress(uint256 userId) external view returns (address) {
        return Clones.predictDeterministicAddress(implementation, _salt(userId));
    }

    function deployWallet(uint256 userId) public onlyRelayer returns (address wallet) {
        address predicted = Clones.predictDeterministicAddress(implementation, _salt(userId));
        if (predicted.code.length > 0) {
            return predicted;
        }
        wallet = Clones.cloneDeterministic(implementation, _salt(userId));
        emit WalletDeployed(userId, wallet);
    }

    function deployAndSweepToken(uint256 userId, address token) external onlyRelayer {
        address wallet = deployWallet(userId);
        Forwarder(payable(wallet)).sweepToken(token);
    }

    function deployAndSweepNative(uint256 userId) external onlyRelayer {
        address wallet = deployWallet(userId);
        Forwarder(payable(wallet)).sweepNative();
    }

    // ─────────────────────────────────────────
    // Emergency functions
    // ─────────────────────────────────────────

    /// @notice Emergency token withdrawal from a specific user wallet — Owner only.
    /// Funds are always forwarded to motherWallet (enforced inside Forwarder).
    function emergencyWithdrawToken(uint256 userId, address token) external onlyOwner {
        address wallet = Clones.predictDeterministicAddress(implementation, _salt(userId));
        require(wallet.code.length > 0, "Factory: wallet not deployed");
        require(token != address(0), "Factory: zero token");
        Forwarder(payable(wallet)).emergencyWithdrawToken(token);
        emit EmergencyWithdraw(userId, token, motherWallet);
    }

    /// @notice Emergency native token withdrawal from a specific user wallet — Owner only.
    /// Funds are always forwarded to motherWallet (enforced inside Forwarder).
    function emergencyWithdrawNative(uint256 userId) external onlyOwner {
        address wallet = Clones.predictDeterministicAddress(implementation, _salt(userId));
        require(wallet.code.length > 0, "Factory: wallet not deployed");
        Forwarder(payable(wallet)).emergencyWithdrawNative();
        emit EmergencyWithdraw(userId, address(0), motherWallet);
    }

    // ─────────────────────────────────────────
    // Admin functions with Timelock
    // ─────────────────────────────────────────

    /// @notice Step 1: request a Mother Wallet change
    /// Must wait 48 hours before it can be applied
    function requestMotherWalletChange(address newMotherWallet) external onlyOwner {
        require(newMotherWallet != address(0), "Factory: zero address");
        pendingMotherWallet = newMotherWallet;
        motherWalletUnlockTime = block.timestamp + TIMELOCK_DELAY;
        emit MotherWalletChangeRequested(newMotherWallet, motherWalletUnlockTime);
    }

    /// @notice Step 2: apply the Mother Wallet change (after 48 hours)
    function applyMotherWalletChange() external onlyOwner {
        require(pendingMotherWallet != address(0), "Factory: no pending change");
        require(block.timestamp >= motherWalletUnlockTime, "Factory: timelock active");
        address old = motherWallet;
        motherWallet = pendingMotherWallet;
        pendingMotherWallet = address(0);
        motherWalletUnlockTime = 0;
        emit MotherWalletUpdated(old, motherWallet);
    }

    /// @notice Cancel a pending Mother Wallet change before it is applied
    function cancelMotherWalletChange() external onlyOwner {
        require(pendingMotherWallet != address(0), "Factory: no pending change");
        address cancelled = pendingMotherWallet;
        pendingMotherWallet = address(0);
        motherWalletUnlockTime = 0;
        emit MotherWalletChangeCancelled(cancelled);
    }

    /// @notice Update the Relayer — no Timelock, lower risk
    function updateRelayer(address newRelayer) external onlyOwner {
        require(newRelayer != address(0), "Factory: zero relayer");
        emit RelayerUpdated(relayer, newRelayer);
        relayer = newRelayer;
    }

    // ─────────────────────────────────────────
    // Two-step Ownership Transfer
    // ─────────────────────────────────────────

    /// @notice Step 1 — current owner nominates a new owner.
    /// Ownership does NOT transfer yet; the new owner must call acceptOwnership().
    /// @param newOwner The address being nominated (e.g. a Gnosis Safe address)
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Factory: zero owner");
        require(newOwner != owner, "Factory: already owner");
        pendingOwner = newOwner;
        emit OwnershipTransferStarted(owner, newOwner);
    }

    /// @notice Step 2 — pending owner accepts and finalizes the transfer.
    /// Must be called by the exact address nominated in transferOwnership().
    /// This confirms the new owner has access to that wallet/key.
    function acceptOwnership() external {
        require(msg.sender == pendingOwner, "Factory: not pending owner");
        address old = owner;
        owner = pendingOwner;
        pendingOwner = address(0);
        emit OwnershipTransferred(old, owner);
    }

    /// @notice Cancel a pending ownership transfer before it is accepted.
    /// Only the current owner can cancel.
    function cancelOwnershipTransfer() external onlyOwner {
        require(pendingOwner != address(0), "Factory: no pending transfer");
        address cancelled = pendingOwner;
        pendingOwner = address(0);
        emit OwnershipTransferCancelled(cancelled);
    }
}
