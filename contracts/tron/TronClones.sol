// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @title TronClones
/// @notice EIP-1167 clone address prediction for TRON TVM.
/// @dev TVM CREATE2 derives addresses with a 0x41 prefix instead of EVM's 0xff.
///      Deployment via OpenZeppelin {Clones-cloneDeterministic} is unchanged on Tron;
///      only off-chain / view prediction must use this library.
library TronClones {
    /// @dev Computes the address of a clone deployed using {Clones-cloneDeterministic} on TVM.
    function predictDeterministicAddress(
        address implementation,
        bytes32 salt,
        address deployer
    ) internal pure returns (address predicted) {
        /// @solidity memory-safe-assembly
        assembly {
            let ptr := mload(0x40)
            mstore(ptr, 0x3d602d80600a3d3981f3363d3d373d3d3d363d73000000000000000000000000)
            mstore(add(ptr, 0x14), shl(0x60, implementation))
            mstore(add(ptr, 0x28), 0x5af43d82803e903d91602b57fd5bf34100000000000000000000000000000000)
            mstore(add(ptr, 0x38), shl(0x60, deployer))
            mstore(add(ptr, 0x4c), salt)
            mstore(add(ptr, 0x6c), keccak256(ptr, 0x37))
            predicted := keccak256(add(ptr, 0x37), 0x55)
        }
    }

    /// @dev Computes the address of a clone deployed using {Clones-cloneDeterministic} on TVM.
    function predictDeterministicAddress(
        address implementation,
        bytes32 salt
    ) internal view returns (address predicted) {
        return predictDeterministicAddress(implementation, salt, address(this));
    }
}
