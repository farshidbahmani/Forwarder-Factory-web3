// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/proxy/Clones.sol";
import "./Forwarder.sol";
import "./tron/TronClones.sol";

/// @title ForwarderFactoryTron
/// @notice Tron TVM variant of ForwarderFactory.
/// @dev Uses {TronClones} for address prediction (0x41 CREATE2 prefix) and
///      OpenZeppelin {Clones} for deployment (TVM create2 opcode handles prefix).
contract ForwarderFactoryTron {
    // NOTE: no longer `immutable`. See constructor + setImplementation() comments
    // below for why, and for why this is still safe.
    address public implementation;

    address public motherWallet;
    address public relayer;
    address public owner;

    address public pendingOwner;

    uint256 public constant TIMELOCK_DELAY = 48 hours;
    address public pendingMotherWallet;
    uint256 public motherWalletUnlockTime;

    event WalletDeployed(uint256 indexed userId, address wallet);
    event RelayerUpdated(address oldRelayer, address newRelayer);
    event MotherWalletChangeRequested(address newMotherWallet, uint256 unlockTime);
    event MotherWalletUpdated(address oldMotherWallet, address newMotherWallet);
    event MotherWalletChangeCancelled(address cancelledAddress);
    event OwnershipTransferStarted(address indexed currentOwner, address indexed pendingOwner);
    event OwnershipTransferred(address indexed oldOwner, address indexed newOwner);
    event OwnershipTransferCancelled(address indexed cancelledPendingOwner);
    event EmergencyWithdraw(uint256 indexed userId, address token, address to);
    event ImplementationSet(address implementation);

    modifier onlyOwner() {
        require(msg.sender == owner, "Factory: not owner");
        _;
    }

    modifier onlyRelayer() {
        require(msg.sender == relayer, "Factory: not relayer");
        _;
    }

    // NOTE: the Factory no longer deploys `Forwarder` inline via `new Forwarder(...)`
    // inside its own constructor. Deploying two contracts (Factory + implementation)
    // in a single transaction is what made the Tron/BNB deployment Energy cost high.
    //
    // Deployment now happens in two steps:
    //   1. Deploy `ForwarderFactoryTron` with this constructor (cheaper — no nested
    //      CREATE).
    //   2. Deploy `Forwarder`, passing this Factory's REAL, now-known address as its
    //      `factory` argument (the Factory already exists on-chain at this point, so
    //      there is no address-prediction risk).
    //   3. Call `setImplementation()` below, once, as the owner.
    //
    // This is deliberately NOT done via a constructor parameter, because that would
    // require predicting the Factory's own future address in advance (from deployer
    // nonce) to deploy `Forwarder` beforehand — fragile, since any other transaction
    // from the deployer account in between would silently change that address and
    // permanently break the system. The two-step setter below avoids that risk
    // entirely at the cost of one extra, cheap transaction.
    constructor(address _motherWallet, address _relayer) {
        require(_motherWallet != address(0), "Factory: zero mother wallet");
        require(_relayer != address(0), "Factory: zero relayer");
        owner = msg.sender;
        motherWallet = _motherWallet;
        relayer = _relayer;
    }

    /// @notice Sets the Forwarder implementation used for all clones. Callable
    /// exactly once, by the owner only. After this call, `implementation` behaves
    /// exactly like an immutable value would have — it can never be changed again,
    /// so no new security surface is introduced by dropping `immutable`.
    function setImplementation(address _implementation) external onlyOwner {
        require(implementation == address(0), "Factory: implementation already set");
        require(_implementation != address(0), "Factory: zero implementation");
        require(_implementation.code.length > 0, "Factory: implementation not deployed");
        implementation = _implementation;
        emit ImplementationSet(_implementation);
    }

    function _salt(uint256 userId) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(userId));
    }

    function getAddress(uint256 userId) external view returns (address) {
        require(implementation != address(0), "Factory: implementation not set");
        return TronClones.predictDeterministicAddress(implementation, _salt(userId));
    }

    function deployWallet(uint256 userId) public onlyRelayer returns (address wallet) {
        require(implementation != address(0), "Factory: implementation not set");
        address predicted = TronClones.predictDeterministicAddress(implementation, _salt(userId));
        if (predicted.code.length > 0) {
            return predicted;
        }
        wallet = Clones.cloneDeterministic(implementation, _salt(userId));
        emit WalletDeployed(userId, wallet);
    }

    function _requireForwarderWallet(address wallet) internal view {
        require(wallet != address(0), "Factory: zero wallet");
        require(wallet.code.length > 0, "Factory: wallet not deployed");
        require(Forwarder(payable(wallet)).factory() == address(this), "Factory: not our wallet");
    }

    function sweepToken(address wallet, address token) external onlyRelayer {
        _requireForwarderWallet(wallet);
        require(token != address(0), "Factory: zero token");
        Forwarder(payable(wallet)).sweepToken(token);
    }

    function sweepNative(address wallet) external onlyRelayer {
        _requireForwarderWallet(wallet);
        Forwarder(payable(wallet)).sweepNative();
    }

    /// @notice Deploys the user's wallet if needed, then sweeps its token balance —
    /// all in one transaction. `deployWallet` is idempotent, so repeat calls only
    /// pay a cheap address prediction + code-size check.
    /// @dev No `_requireForwarderWallet` needed: the address is derived from
    /// `userId` via CREATE2, so it can only ever be our own clone.
    function deployAndSweepToken(uint256 userId, address token) external onlyRelayer {
        require(token != address(0), "Factory: zero token");
        address wallet = deployWallet(userId);
        Forwarder(payable(wallet)).sweepToken(token);
    }

    /// @notice Deploys the user's wallet if needed, then sweeps its native balance —
    /// all in one transaction. See {deployAndSweepToken} for details.
    function deployAndSweepNative(uint256 userId) external onlyRelayer {
        address wallet = deployWallet(userId);
        Forwarder(payable(wallet)).sweepNative();
    }

    function emergencyWithdrawToken(uint256 userId, address token) external onlyOwner {
        address wallet = TronClones.predictDeterministicAddress(implementation, _salt(userId));
        require(wallet.code.length > 0, "Factory: wallet not deployed");
        require(token != address(0), "Factory: zero token");
        Forwarder(payable(wallet)).emergencyWithdrawToken(token);
        emit EmergencyWithdraw(userId, token, motherWallet);
    }

    function emergencyWithdrawNative(uint256 userId) external onlyOwner {
        address wallet = TronClones.predictDeterministicAddress(implementation, _salt(userId));
        require(wallet.code.length > 0, "Factory: wallet not deployed");
        Forwarder(payable(wallet)).emergencyWithdrawNative();
        emit EmergencyWithdraw(userId, address(0), motherWallet);
    }

    function requestMotherWalletChange(address newMotherWallet) external onlyOwner {
        require(newMotherWallet != address(0), "Factory: zero address");
        pendingMotherWallet = newMotherWallet;
        motherWalletUnlockTime = block.timestamp + TIMELOCK_DELAY;
        emit MotherWalletChangeRequested(newMotherWallet, motherWalletUnlockTime);
    }

    function applyMotherWalletChange() external onlyOwner {
        require(pendingMotherWallet != address(0), "Factory: no pending change");
        require(block.timestamp >= motherWalletUnlockTime, "Factory: timelock active");
        address old = motherWallet;
        motherWallet = pendingMotherWallet;
        pendingMotherWallet = address(0);
        motherWalletUnlockTime = 0;
        emit MotherWalletUpdated(old, motherWallet);
    }

    function cancelMotherWalletChange() external onlyOwner {
        require(pendingMotherWallet != address(0), "Factory: no pending change");
        address cancelled = pendingMotherWallet;
        pendingMotherWallet = address(0);
        motherWalletUnlockTime = 0;
        emit MotherWalletChangeCancelled(cancelled);
    }

    function updateRelayer(address newRelayer) external onlyOwner {
        require(newRelayer != address(0), "Factory: zero relayer");
        emit RelayerUpdated(relayer, newRelayer);
        relayer = newRelayer;
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Factory: zero owner");
        require(newOwner != owner, "Factory: already owner");
        pendingOwner = newOwner;
        emit OwnershipTransferStarted(owner, newOwner);
    }

    function acceptOwnership() external {
        require(msg.sender == pendingOwner, "Factory: not pending owner");
        address old = owner;
        owner = pendingOwner;
        pendingOwner = address(0);
        emit OwnershipTransferred(old, owner);
    }

    function cancelOwnershipTransfer() external onlyOwner {
        require(pendingOwner != address(0), "Factory: no pending transfer");
        address cancelled = pendingOwner;
        pendingOwner = address(0);
        emit OwnershipTransferCancelled(cancelled);
    }
}
