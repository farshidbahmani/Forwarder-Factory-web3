// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/// @notice A simple mintable BEP20 token for local testing only.
/// @dev NEVER deploy this on mainnet — the mint function is publicly accessible.
contract MockBEP20 is ERC20 {
    constructor(string memory name, string memory symbol) ERC20(name, symbol) {}

    /// @notice Mints tokens to any address. For test purposes only.
    function mint(address to, uint256 amount) external {
        _mint(to, amount);
    }
}
