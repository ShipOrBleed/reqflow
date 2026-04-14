package reqflow

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"
)

// noiseFields are struct field names that are infrastructure, not request flow.
var noiseFields = map[string]bool{
	"Logger": true, "logger": true, "log": true, "Log": true,
	"Metrics": true, "metrics": true, "Metric": true,
	"Tracer": true, "tracer": true,
	"Context": true, "Ctx": true, "ctx": true,
	"mu": true, "Mutex": true, "mutex": true,
	"wg": true, "WaitGroup": true,
	"once": true, "Once": true,
	"timer": true, "Timer": true, "ticker": true,
}

// noiseMethods are method names that are infrastructure calls.
var noiseMethods = map[string]bool{
	"Errorf": true, "Infof": true, "Warnf": true, "Debugf": true, "Printf": true,
	"Error": true, "Info": true, "Warn": true, "Debug": true, "Print": true, "Println": true,
	"Fatalf": true, "Fatal": true, "Panicf": true, "Panic": true,
	"Log": true, "Logf": true,
	"WithField": true, "WithFields": true, "WithError": true, "WithContext": true,
	"Lock": true, "Unlock": true, "RLock": true, "RUnlock": true,
	"Add": true, "Done": true, "Wait": true,
	"IncrementCounter": true, "RecordHistogram": true, "SetGauge": true,
	"Start": true, "End": true, "Span": true,
}

func isNoiseCall(fieldName, methodName string) bool {
	return noiseFields[fieldName] || noiseMethods[methodName]
}

// MethodCall represents a call from one struct method to another struct's method.
// Example: Handler.GetMetrics calls field "svc" method "GetMetrics" on Service.
type MethodCall struct {
	FieldName    string // e.g. "svc", "store"
	TargetMethod string // e.g. "GetMetrics", "GetCostRecords"
}

// MethodCallIndex maps "pkg.Struct.Method" → list of outgoing method calls.
type MethodCallIndex map[string][]MethodCall

// buildMethodCallIndex walks all receiver methods and finds calls of the form:
//
//	receiver.field.Method(...)
//
// This allows the trace to show only the specific methods called at each layer,
// not all methods on the struct.
func buildMethodCallIndex(pkgs []*packages.Package, graph *Graph) MethodCallIndex {
	index := make(MethodCallIndex)

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || fn.Body == nil {
					return true
				}

				// Get receiver struct ID
				recvType := fn.Recv.List[0].Type
				if star, ok := recvType.(*ast.StarExpr); ok {
					recvType = star.X
				}
				ident, ok := recvType.(*ast.Ident)
				if !ok {
					return true
				}
				structID := pkg.PkgPath + "." + ident.Name
				methodKey := structID + "." + fn.Name.Name

				// Walk the body looking for field.Method() calls
				ast.Inspect(fn.Body, func(inner ast.Node) bool {
					call, ok := inner.(*ast.CallExpr)
					if !ok {
						return true
					}

					// Pattern: h.svc.GetMetrics() → SelectorExpr(SelectorExpr(Ident, field), method)
					outerSel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					methodName := outerSel.Sel.Name

					innerSel, ok := outerSel.X.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					fieldName := innerSel.Sel.Name

					// Skip noise: logging, metrics, context helpers
					if isNoiseCall(fieldName, methodName) {
						return true
					}

					index[methodKey] = append(index[methodKey], MethodCall{
						FieldName:    fieldName,
						TargetMethod: methodName,
					})

					return true
				})

				return true
			})
		}
	}

	return index
}

// resolveCalledMethods finds which methods on a target node are actually called
// by a given source method. It matches field names from the call index against
// the node's fields to determine which struct's methods are being invoked.
func resolveCalledMethods(index MethodCallIndex, sourceMethodKey string, targetNode *Node) []string {
	calls := index[sourceMethodKey]
	if len(calls) == 0 {
		return nil
	}

	var methods []string
	seen := make(map[string]bool)

	for _, call := range calls {
		// Check if the target node matches any field on the source struct
		// by seeing if call.TargetMethod is a method on targetNode
		for _, m := range targetNode.Methods {
			if m == call.TargetMethod && !seen[m] {
				seen[m] = true
				methods = append(methods, m)
			}
		}
	}

	return methods
}

// getCalledMethodsOnNode returns the specific methods called on targetNode
// when the source method is known. Falls back to showing all methods if
// the call chain can't be resolved.
func getCalledMethodsOnNode(index MethodCallIndex, sourceStructID, sourceMethod string, targetNode *Node) []string {
	if sourceMethod == "" || index == nil {
		return nil
	}

	key := sourceStructID + "." + sourceMethod
	methods := resolveCalledMethods(index, key, targetNode)

	// If we found specific calls, also look one level deeper:
	// the target method may call methods on its own fields
	if len(methods) == 0 {
		// Try matching all calls from source method against target methods
		calls := index[key]
		for _, call := range calls {
			for _, m := range targetNode.Methods {
				if strings.EqualFold(call.TargetMethod, m) {
					methods = append(methods, m)
				}
			}
		}
	}

	return methods
}
