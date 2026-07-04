import { ethers } from "ethers";
import {
  ContractCallResult,
  ContractFunctionDef,
  FORWARDER_FACTORY_FUNCTIONS,
} from "../../domain/contract";
import { AppError } from "../../domain/errors";
import { envKeyForNetwork, getNetwork } from "../../domain/network";
import { EnvRepository } from "../env/env.repository";
import { ProviderFactory } from "./provider.factory";

function isEthersResult(value: unknown): value is ethers.Result {
  return (
    value != null &&
    typeof value === "object" &&
    "toArray" in value &&
    typeof (value as { toArray: unknown }).toArray === "function"
  );
}

function toResultArray(result: unknown): unknown[] {
  if (isEthersResult(result)) return result.toArray();
  if (Array.isArray(result)) return result;
  return [result];
}

export class ContractGateway {
  constructor(
    private readonly envRepo: EnvRepository,
    private readonly providerFactory: ProviderFactory,
  ) {}

  listFunctions(): ContractFunctionDef[] {
    return FORWARDER_FACTORY_FUNCTIONS;
  }

  private resolveFactoryAddress(networkName: string): string {
    const network = getNetwork(networkName);
    const key = envKeyForNetwork("FACTORY_ADDRESS", network);

    const address = this.envRepo.get(key);
    if (!address) {
      throw new AppError(`No factory deployed. Set ${key} in .env`);
    }
    if (!ethers.isAddress(address)) {
      throw new AppError(`Invalid ${key} in .env`);
    }
    return ethers.getAddress(address);
  }

  private async assertContractDeployed(networkName: string, address: string): Promise<void> {
    const { provider, network } = this.providerFactory.getProvider(networkName);
    const code = await provider.getCode(address);
    if (code === "0x") {
      throw new AppError(
        `No contract at ${address} on ${network.name}. ` +
          `Check FACTORY_ADDRESS_${network.envSuffix} — it may be a wallet address, not the deployed factory.`,
      );
    }
  }

  private parseArgs(fn: ContractFunctionDef, rawArgs: Record<string, string>): unknown[] {
    return fn.inputs.map((input) => {
      const value = rawArgs[input.name]?.trim();
      if (!value) throw new AppError(`Missing parameter: ${input.label}`);
      if (input.type === "uint256") return BigInt(value);
      if (input.type === "address") {
        if (!ethers.isAddress(value)) throw new AppError(`Invalid address: ${input.label}`);
        return ethers.getAddress(value);
      }
      return value;
    });
  }

  private formatValue(value: unknown, solidityType?: string): unknown {
    if (value === null || value === undefined) return value;

    if (isEthersResult(value)) {
      return value.toArray().map((v) => this.formatValue(v));
    }

    if (typeof value === "bigint") return value.toString();

    if (
      solidityType === "address" ||
      (typeof value === "string" && ethers.isAddress(value))
    ) {
      return ethers.getAddress(value as string);
    }

    if (Array.isArray(value)) {
      return value.map((v) => this.formatValue(v));
    }

    return value;
  }

  private formatCallResult(
    fragment: ethers.FunctionFragment,
    result: unknown,
  ): unknown {
    const values = toResultArray(result);
    const outputs = fragment.outputs;

    if (outputs.length === 0) return null;

    if (outputs.length === 1) {
      return this.formatValue(values[0], outputs[0].type);
    }

    const named: Record<string, unknown> = {};
    outputs.forEach((output, index) => {
      const key = output.name || String(index);
      named[key] = this.formatValue(values[index], output.type);
    });
    return named;
  }

  async call(
    networkName: string,
    functionName: string,
    args: Record<string, string>,
  ): Promise<ContractCallResult> {
    const fn = FORWARDER_FACTORY_FUNCTIONS.find((f) => f.name === functionName);
    if (!fn) throw new AppError(`Unknown function: ${functionName}`);

    const factoryAddress = this.resolveFactoryAddress(networkName);
    await this.assertContractDeployed(networkName, factoryAddress);
    const parsedArgs = this.parseArgs(fn, args);

    if (fn.type === "read") {
      const contract = this.providerFactory.getFactoryContract(networkName, factoryAddress);
      const method = contract.getFunction(fn.name);
      const result = await method.staticCall(...parsedArgs);
      return {
        functionName,
        type: "read",
        result: this.formatCallResult(method.fragment, result),
      };
    }

    const role = fn.role === "relayer" ? "relayer" : "owner";
    const signer = this.providerFactory.getSigner(networkName, role);
    const contract = this.providerFactory.getFactoryContract(networkName, factoryAddress, signer);
    const tx = await contract.getFunction(fn.name)(...parsedArgs);
    const receipt = await tx.wait();

    return {
      functionName,
      type: "write",
      txHash: receipt?.hash,
      blockNumber: receipt?.blockNumber,
      gasUsed: receipt?.gasUsed?.toString(),
    };
  }

  async getFactoryInfo(networkName: string) {
    const factoryAddress = this.resolveFactoryAddress(networkName);
    await this.assertContractDeployed(networkName, factoryAddress);
    const contract = this.providerFactory.getFactoryContract(networkName, factoryAddress);
    const [motherWallet, relayer, owner, implementation] = await Promise.all([
      contract.motherWallet(),
      contract.relayer(),
      contract.owner(),
      contract.implementation(),
    ]);
    return { factoryAddress, motherWallet, relayer, owner, implementation };
  }
}
