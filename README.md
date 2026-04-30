# TS Inspector

A static analyser and LSP server for parsing, analysing, and manipulating TypeScript projects, with a focus on supporting Angular projects using Pug templates.

## Features

### Analysis + Diagnostics

TS Inspector analyses your code to find potential problems

#### Angular
- Illegal Decorator Combinations: Detects when a class has been illegally decorated with both `@Component` and `@Module`.
- Lifecycle Hooks Without Implements Clause: Detects when a component uses an Angular lifecycle hook (`ngOnInit` and friends), without the relevant `implements` class on the class.
- Async Lifecycle Hooks: Detects when lifecycle hooks are illegally async.
- Incorrectly Private Angular Properties: Detects when `@Input`s, `@Output`s, etc. are private when they should be public.
- Unused Angular Properties: Detects when an `@Input`, `@Output`, etc. is unused.
- Recursive Templates: Highlights when a component recursively uses itself in its template.
- Getters Used In Templates: Highlights possible performance problems when a getter is used in a component's template.
- (WIP) Type-checked Pug templates: detects when the wrong types are used for bindings, e.g., trying to pass a `string` to a `number` binding.

#### TypeScript
- Constructor Only Property: Highlights when constructor arguments have needlessly been made into class properties.
- Unnecessarily Public Properties: Detects when a class property needn't be public.
- Illegal Module Declarations: Detects when an Angular module is illegally declaring another Angular module.
- Unreachable Code: Highlights unreachable sections of code through the use of control flow graph analysis.

### Code Actions

Code actions are available to make refactorings and perform tasks

- Lifecycle Hook Skeletons: Adds skeleton Angular lifecycle hook implementations (`OnInit`, `OnDestroy`, and friends) to the current class, including class `implements` clause and imports.
- Dependency Injection Migration: Converts an `@Inject(token) public prop: type` property to a `public prop: type = inject(token)`, including updating imports.
- Add Component Destruction Observable: Adds a `_destroyed$` observable, which can be used in RxJS pipes to stop subscriptions when the component is destroyed.
- Make Surrounding Function Async: Makes the surrounding function/method/arrow function async with `Promise<T>` return type and `async` keyword.
- Rearrange Class: Rearranges methods and properties in the class.
- Go To Declaring Module: Navigate to the Angular module where the current component is declared, if it exists.
- Calculate All Providers: Prints a list of all of the providers available to the current component. Useful if you have configuration options being resolved from providers.

### Go To Definition

Supports go to definition from tags in templates to their component class definitions, as well as properties in Angular bindings in template files.

### Find References

From a template, finds all of the usages of a component from its selector

### Code Completion

Supports context-aware completions for component selectors, `[input]`s, `(output)`s, class properties + methods.

### Hover

Shows relevant and useful information, like `[input]`s, `(output)`s, and declaring class when hovering tag names in templates, as well as type information for properties in Angular bindings.
