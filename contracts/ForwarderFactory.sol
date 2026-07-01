// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/proxy/Clones.sol";
import "./Forwarder.sol";

/// @title ForwarderFactory
/// @notice Additional protections:
///   - Timelock: motherWallet changes must be announced 48 hours in advance
///   - emergencyWithdraw: Owner can rescue funds from any user wallet directly
///   - Events emitted for all important state changes
contract ForwarderFactory {
    address public immutable implementation;

    address public motherWallet;
    address public relayer;
    address public owner;

    // ─── Timelock for Mother Wallet changes ───
    uint256 public constant TIMELOCK_DELAY = 48 hours;
    address public pendingMotherWallet;
    uint256 public motherWalletUnlockTime;

    event WalletDeployed(uint256 indexed userId, address wallet);
    event RelayerUpdated(address oldRelayer, address newRelayer);
    event MotherWalletChangeRequested(address newMotherWallet, uint256 unlockTime);
    event MotherWalletUpdated(address oldMotherWallet, address newMotherWallet);
    event MotherWalletChangeCancelled(address cancelledAddress);
    event OwnerUpdated(address oldOwner, address newOwner);
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

    function deployAndSweepBNB(uint256 userId) external onlyRelayer {
        address wallet = deployWallet(userId);
        Forwarder(payable(wallet)).sweepBNB();
    }

    // ─────────────────────────────────────────
    // Emergency functions
    // ─────────────────────────────────────────

    /// @notice Emergency token withdrawal from a specific user wallet — Owner only
    function emergencyWithdrawToken(
        uint256 userId,
        address token,
        address to
    ) external onlyOwner {
        address wallet = Clones.predictDeterministicAddress(implementation, _salt(userId));
        require(wallet.code.length > 0, "Factory: wallet not deployed");
        Forwarder(payable(wallet)).emergencyWithdrawToken(token, to);
        emit EmergencyWithdraw(userId, token, to);
    }

    /// @notice Emergency BNB withdrawal from a specific user wallet — Owner only
    function emergencyWithdrawBNB(uint256 userId, address to) external onlyOwner {
        address wallet = Clones.predictDeterministicAddress(implementation, _salt(userId));
        require(wallet.code.length > 0, "Factory: wallet not deployed");
        Forwarder(payable(wallet)).emergencyWithdrawBNB(to);
        emit EmergencyWithdraw(userId, address(0), to);
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

    /// @notice Transfer ownership — ideally to a Multisig
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Factory: zero owner");
        emit OwnerUpdated(owner, newOwner);
        owner = newOwner;
    }
}
