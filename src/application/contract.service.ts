import { ContractCallResult, ContractFunctionDef } from "../domain/contract";
import { ContractGateway } from "../infrastructure/blockchain/contract.gateway";

export class ContractService {
  constructor(private readonly gateway: ContractGateway) {}

  listFunctions(): ContractFunctionDef[] {
    return this.gateway.listFunctions();
  }

  async call(
    networkName: string,
    functionName: string,
    args: Record<string, string>,
  ): Promise<ContractCallResult> {
    return this.gateway.call(networkName, functionName, args);
  }

  async getFactoryInfo(networkName: string) {
    return this.gateway.getFactoryInfo(networkName);
  }
}
