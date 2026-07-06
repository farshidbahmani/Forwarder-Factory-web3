package contract

import "forwarder-factory/internal/factoryabi"

type Param = factoryabi.Param
type FunctionDef = factoryabi.FunctionDef
type CallResult = factoryabi.CallResult
type FactoryInfo = factoryabi.FactoryInfo

var Functions = factoryabi.Functions

func FindFunction(name string) (FunctionDef, bool) {
	return factoryabi.FindFunction(name)
}

func ListFunctions() []FunctionDef { return Functions }
