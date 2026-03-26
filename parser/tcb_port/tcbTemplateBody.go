package tcb_port

// import (
// 	"fmt"
// 	"slices"
// 	"ts_inspector/parser/ast"
// )

type TcbTemplateBodyOp struct {
	TcbOp
	tcb      Context
	scope    Scope
	template TmplAstNode
}

func (o TcbTemplateBodyOp) Optional() bool { return false }
func (o TcbTemplateBodyOp) Execute() *Identifier {
	// 	// An `if` will be constructed, within which the template's children will be type checked. The
	// 	// `if` is used for two reasons: it creates a new syntactic scope, isolating variables declared
	// 	// in the template's TCB from the outer context, and it allows any directives on the templates
	// 	// to perform type narrowing of either expressions or the template's context.
	// 	//
	// 	// The guard is the `if` block's condition. It's usually set to `true` but directives that exist
	// 	// on the template can trigger extra guard expressions that serve to narrow types within the
	// 	// `if`. `guard` is calculated by starting with `true` and adding other conditions as needed.
	// 	// Collect these into `guards` by processing the directives.
	// 	directiveGuards := Expression{}
	//
	// 	// directives := tcb.boundTarget.getDirectivesOfNode(this.template)
	// 	// directives :=
	//
	// 	for dir, _ := range directives {
	// 		dirInstId := scope.resolve(template, dir)
	// 		dirId := tcb.env.reference(dir.typeScriptClass)
	//
	// 		// There are two kinds of guards. Template guards (ngTemplateGuards) allow type narrowing of
	// 		// the expression passed to an @Input of the directive. Scan the directive to see if it has
	// 		// any template guards, and generate them if needed.
	// 		for guard, _ := range dir.templateGuards {
	// 			// For each template guard function on the directive, look for a binding to that input.
	// 			boundInput := template.inputs[guard.inputName]
	// 			if boundInput == nil {
	// 				for attr, _ := range template.templateAttrs {
	// 					if attr.name == guard.name {
	// 						boundInput = attr
	// 						break
	// 					}
	// 				}
	// 			}
	//
	// 			if boundInput != nil {
	// 				// If there is such a binding, generate an expression for it.
	// 				expr := tcbExpression(boundInput.value, tcb, scope)
	//
	// 				if guard.ttype == Angular2TemplateGuard.Kind.Binding {
	// 					// Use the binding expression itself as guard.
	// 					directiveGuards = append(directiveGuards, expr)
	// 				} else {
	// 					// Call the guard function on the directive with the directive instance and that
	// 					// guardInvoke.
	// 					guardInvoke := Expression{}
	// 					guardInvoke = append(guardInvoke, "$dirId.$NG_TEMPLATE_GUARD_PREFIX${guard.inputName}")
	// 				guardInvoke = append(guardInvoke, fmt.Sprintf("%s.%s%s", dirId, NG_TEMPLATE_CONTEXT_GUARD, guard.inputName))
	// 					guardInvoke = append(guardInvoke, "($dirInstId, ")
	// 					guardInvoke = append(guardInvoke, expr)
	// 					guardInvoke = append(guardInvoke, ")")
	//
	// 					directiveGuards = append(directiveGuards, guardInvoke...)
	// 				}
	// 			}
	// 		}
	//
	// 		// The second kind of guard is a template context guard. This guard narrows the template
	// 		// rendering context variable `ctx`.
	// 		if dir.hasTemplateContextGuard {
	// 			suggestionsForSuboptimalTypeInference := true // this.tcb.env.config.suggestionsForSuboptimalTypeInference
	//
	// 			if this.tcb.env.config.applyTemplateContextGuards {
	// 				ctx := scope.resolve(template)
	//
	// 				guardInvoke := Expression{}
	// 				guardInvoke = append(guardInvoke, fmt.Sprintf("%s.%s(%s, %s)", dirId, NG_TEMPLATE_CONTEXT_GUARD, dirInstId, ctx))
	//
	// 				directiveGuards.add(guardInvoke)
	// 			} else if len(template.variables) > 0 && suggestionsForSuboptimalTypeInference {
	// 				// The compiler could have inferred a better type for the variables in this template,
	// 				// but was prevented from doing so by the type-checking configuration. Issue a warning
	// 				// diagnostic.
	//
	// 				// TODO CM: figure out what to do
	// 				// this.tcb.oobRecorder.suboptimalTypeInference(this.tcb.id, this.template.variables.values)
	// 			}
	// 		}
	// 	}
	//
	// 	// By default the guard is simply `true`.
	// 	var guard *Expression = nil
	//
	// 	// If there are any guards from directives, use them instead.
	// 	if len(directiveGuards) > 0 {
	// 		// Pop the first value and use it as the initializer to reduce(). This way, a single guard
	// 		// will be used on its own, but two or more will be combined into binary AND expressions.
	// 		guard = Expression{}
	// 		for index, expression := range slices.Backward(directiveGuards) {
	// 			if index > 0 {
	// 				guard = append(guard, " && ")
	// 			}
	// 			guard = append(guard, "(")
	// 			guard = append(guard, expression)
	// 			guard = append(guard, ")")
	// 		}
	// 	}
	//
	// 	// Create a new Scope for the template. This constructs the list of operations for the template
	// 	// children, as well as tracks bindings within the template.
	// 	tmplScope := scopeForNodes(tcb, scope, template, template.children, guard)
	//
	// 	// Render the template's `Scope` into its statements.
	// 	statements := tmplScope.render()
	// 	if len(statements) > 0 {
	// 		// As an optimization, don't generate the scope's block if it has no statements. This is
	// 		// beneficial for templates that contain for example `<span *ngIf="first"></span>`, in which
	// 		// case there's no need to render the `NgIf` guard expression. This seems like a minor
	// 		// improvement, however it reduces the number of flow-node antecedents that TypeScript needs
	// 		// to keep into account for such cases, resulting in an overall reduction of
	// 		// type-checking time.
	// 		return nil
	// 	}
	//
	// 	statement := Statement{}
	// 	if guard != nil {
	// 		// The scope has a guard that needs to be applied, so wrap the template block into an `if`
	// 		// statement containing the guard expression.
	// 		s := &statement
	// 		statement.AddPart("if (")
	// 		statement.AddPart(guard)
	// 		statement.AddPart(") ")
	// 	}
	//
	// 	statement.AddCodeBlock(func() {
	// 		ctx := scope.resolve(template)
	// 		statement.AddPart(ctx)
	// 		statement.AddPart(";")
	//
	// 		for statement, _ := range statements {
	// 			statement.appendStatement(statement)
	// 		}
	// 	})
	//
	// 	return nil
	//

	return nil
}

// /**
//  * A `TcbOp` which descends into a `TmplAstTemplate`'s children and generates type-checking code for
//  * them.
//  *
//  * This operation wraps the children's type-checking code in an `if` block, which may include one
//  * or more type guard conditions that narrow types within the template body.
//  */
// func handleTemplateBody(tcb *Tcb, scope *Scope, template *ast.Node) Identifier {
// 	// An `if` will be constructed, within which the template's children will be type checked. The
// 	// `if` is used for two reasons: it creates a new syntactic scope, isolating variables declared
// 	// in the template's TCB from the outer context, and it allows any directives on the templates
// 	// to perform type narrowing of either expressions or the template's context.
// 	//
// 	// The guard is the `if` block's condition. It's usually set to `true` but directives that exist
// 	// on the template can trigger extra guard expressions that serve to narrow types within the
// 	// `if`. `guard` is calculated by starting with `true` and adding other conditions as needed.
// 	// Collect these into `guards` by processing the directives.
// 	directiveGuards := Expression{}
//
// 	// directives := tcb.boundTarget.getDirectivesOfNode(this.template)
// 	// directives :=
//
// 	for dir, _ := range directives {
// 		dirInstId := scope.resolve(template, dir)
// 		dirId := tcb.env.reference(dir.typeScriptClass)
//
// 		// There are two kinds of guards. Template guards (ngTemplateGuards) allow type narrowing of
// 		// the expression passed to an @Input of the directive. Scan the directive to see if it has
// 		// any template guards, and generate them if needed.
// 		for guard, _ := range dir.templateGuards {
// 			// For each template guard function on the directive, look for a binding to that input.
// 			boundInput := template.inputs[guard.inputName]
// 			if boundInput == nil {
// 				for attr, _ := range template.templateAttrs {
// 					if attr.name == guard.name {
// 						boundInput = attr
// 						break
// 					}
// 				}
// 			}
//
// 			if boundInput != nil {
// 				// If there is such a binding, generate an expression for it.
// 				expr := tcbExpression(boundInput.value, tcb, scope)
//
// 				if guard.ttype == Angular2TemplateGuard.Kind.Binding {
// 					// Use the binding expression itself as guard.
// 					directiveGuards = append(directiveGuards, expr)
// 				} else {
// 					// Call the guard function on the directive with the directive instance and that
// 					// guardInvoke.
// 					guardInvoke := Expression{}
// 					guardInvoke = append(guardInvoke, "$dirId.$NG_TEMPLATE_GUARD_PREFIX${guard.inputName}")
// 				guardInvoke = append(guardInvoke, fmt.Sprintf("%s.%s%s", dirId, NG_TEMPLATE_CONTEXT_GUARD, guard.inputName))
// 					guardInvoke = append(guardInvoke, "($dirInstId, ")
// 					guardInvoke = append(guardInvoke, expr)
// 					guardInvoke = append(guardInvoke, ")")
//
// 					directiveGuards = append(directiveGuards, guardInvoke...)
// 				}
// 			}
// 		}
//
// 		// The second kind of guard is a template context guard. This guard narrows the template
// 		// rendering context variable `ctx`.
// 		if dir.hasTemplateContextGuard {
// 			suggestionsForSuboptimalTypeInference := true // this.tcb.env.config.suggestionsForSuboptimalTypeInference
//
// 			if this.tcb.env.config.applyTemplateContextGuards {
// 				ctx := scope.resolve(template)
//
// 				guardInvoke := Expression{}
// 				guardInvoke = append(guardInvoke, fmt.Sprintf("%s.%s(%s, %s)", dirId, NG_TEMPLATE_CONTEXT_GUARD, dirInstId, ctx))
//
// 				directiveGuards.add(guardInvoke)
// 			} else if len(template.variables) > 0 && suggestionsForSuboptimalTypeInference {
// 				// The compiler could have inferred a better type for the variables in this template,
// 				// but was prevented from doing so by the type-checking configuration. Issue a warning
// 				// diagnostic.
//
// 				// TODO CM: figure out what to do
// 				// this.tcb.oobRecorder.suboptimalTypeInference(this.tcb.id, this.template.variables.values)
// 			}
// 		}
// 	}
//
// 	// By default the guard is simply `true`.
// 	var guard *Expression = nil
//
// 	// If there are any guards from directives, use them instead.
// 	if len(directiveGuards) > 0 {
// 		// Pop the first value and use it as the initializer to reduce(). This way, a single guard
// 		// will be used on its own, but two or more will be combined into binary AND expressions.
// 		guard = Expression{}
// 		for index, expression := range slices.Backward(directiveGuards) {
// 			if index > 0 {
// 				guard = append(guard, " && ")
// 			}
// 			guard = append(guard, "(")
// 			guard = append(guard, expression)
// 			guard = append(guard, ")")
// 		}
// 	}
//
// 	// Create a new Scope for the template. This constructs the list of operations for the template
// 	// children, as well as tracks bindings within the template.
// 	tmplScope := scopeForNodes(tcb, scope, template, template.children, guard)
//
// 	// Render the template's `Scope` into its statements.
// 	statements := tmplScope.render()
// 	if len(statements) > 0 {
// 		// As an optimization, don't generate the scope's block if it has no statements. This is
// 		// beneficial for templates that contain for example `<span *ngIf="first"></span>`, in which
// 		// case there's no need to render the `NgIf` guard expression. This seems like a minor
// 		// improvement, however it reduces the number of flow-node antecedents that TypeScript needs
// 		// to keep into account for such cases, resulting in an overall reduction of
// 		// type-checking time.
// 		return nil
// 	}
//
// 	statement := Statement{}
// 	if guard != nil {
// 		// The scope has a guard that needs to be applied, so wrap the template block into an `if`
// 		// statement containing the guard expression.
// 		s := &statement
// 		statement.AddPart("if (")
// 		statement.AddPart(guard)
// 		statement.AddPart(") ")
// 	}
//
// 	statement.AddCodeBlock(func() {
// 		ctx := scope.resolve(template)
// 		statement.AddPart(ctx)
// 		statement.AddPart(";")
//
// 		for statement, _ := range statements {
// 			statement.appendStatement(statement)
// 		}
// 	})
//
// 	return nil
// }
