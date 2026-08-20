package parser

import (
	"cmp"
	"slices"
	"strings"
	"sync"
	"ts_inspector/interfaces"

	sitter "github.com/smacker/go-tree-sitter"
)

type Decorator struct {
	Arguments []string // Arguments should have "quotes" or 'quotes' on them
	IsAngular bool
	Name      string
}

type classState struct {
	Angular              *Angular
	Content              string
	Definitions          Definitions
	Extends              References // Extends may have nil references if resolution failed
	ExtendsIdentNames    []string
	File                 *File
	Implements           References // Implements may have nil references if resolution failed
	ImplementsIdentNames []string
	Name                 string
	NameNode             *sitter.Node
	Node                 *sitter.Node
	TypeParameters       []string
	Usages               Usages
}

type Class struct {
	sync.RWMutex
	state classState
}

type GetProvidersResult struct {
	Provider *Provider
	Source   *Class
}

func (c *Class) AddDefinition(definition Definition) {
	if definition.Usages == nil {
		definition.Usages = []*UsageInstance{}
	}

	name := definition.Name
	definition.IsAngularesqueMethod = IsAngularFunction(name)

	c.Update(func(data *classState) {
		if data.Definitions == nil {
			data.Definitions = make(map[string]*Definition)
		}

		data.Definitions[name] = &definition
	})
}

func (c *Class) AppendDefinitionUsage(name string, usage *UsageInstance) {
	definition, found := c.Snapshot().Definitions[name]
	if !found {
		return
	}

	definition.UsageAccess = CalculateNewAccessType(definition.UsageAccess, usage.Access)
	definition.Usages = append(definition.Usages, usage)

	c.Update(func(data *classState) {
		data.Definitions[name] = definition
	})
}

func (c *Class) AppendUsage(name string, usage *UsageInstance) {
	usages, found := c.Snapshot().Usages[name]

	if found {
		usages.Usages = append(usages.Usages, usage)

		c.Update(func(data *classState) {
			data.Usages[name] = usages
		})

		return
	}

	if c.Snapshot().Usages == nil {
		c.Update(func(data *classState) {
			data.Usages = make(map[string]Usage)
		})
	}

	c.Update(func(data *classState) {
		data.Usages[name] = Usage{usage.Access, name, []*UsageInstance{usage}}
	})
}

func (c *Class) DoesExtendOrImplement(class *Class) bool {
	for e := range c.Snapshot().Extends.IterateResolved {
		if e.Class == class {
			return true
		}

		if e.Class.DoesExtendOrImplement(class) {
			return true
		}
	}

	for i := range c.Snapshot().Implements.IterateResolved {
		if i.Class == class {
			return true
		}

		if i.Class.DoesExtendOrImplement(class) {
			return true
		}
	}

	return false
}

func (c *Class) DropTemplateUsages() {
	for usageIndex, usage := range c.Snapshot().Usages {
		usageInstances := make([]*UsageInstance, 0)
		access := NoAccess

		for usageInstanceIndex := range usage.Usages {
			usageInstance := usage.Usages[usageInstanceIndex]

			if usageInstance.Access != TemplateAccess {
				usageInstances = append(usageInstances, usageInstance)
				access = CalculateNewAccessType(access, usageInstance.Access)
			}
		}

		usage.Usages = usageInstances
		usage.Access = access

		c.Update(func(data *classState) {
			data.Usages[usageIndex] = usage
		})

		for definitionIndex, definition := range c.Snapshot().Definitions {
			if definition.Name == usage.Name {
				definition.Usages = usageInstances
				definition.UsageAccess = access
			}

			c.Update(func(data *classState) {
				data.Definitions[definitionIndex] = definition
			})
		}
	}

	for key, usage := range c.Snapshot().Usages {
		if len(usage.Usages) == 0 {

			c.Update(func(data *classState) {
				delete(data.Usages, key)
			})
		}
	}

	if c.HasComponent() && c.Snapshot().Angular.Component.Template != nil {
		c.Snapshot().Angular.Component.Template.TagUsages = make(TagUsages)
	}
}

func (c *Class) EnsureAngular() {
	if c.Snapshot().Angular == nil {
		c.Update(func(data *classState) {
			data.Angular = &Angular{}
		})
	}
}

type ClassedDefinition struct {
	*Definition
	Class *Class
}

func (c *ClassedDefinition) GetDocumentation(includeDefinitionName bool) string {
	documentation := ""
	if includeDefinitionName {
		documentation += "# " + c.Class.Snapshot().Name + "." + c.Name + "\n\n"
	}

	return documentation + c.Definition.GetDocumentation(false)
}

func (c *ClassedDefinition) GetLocation() interfaces.Location {
	classStart := c.Class.Snapshot().Node.StartByte()
	node := c.GetNameNode()

	start := node.StartByte() + classStart
	end := node.EndByte() + classStart

	return c.Class.Snapshot().File.GetLocationForOffset(start, end)
}

func (c *Class) GetInterestingPoints() []InterestingPoint {
	interestingPoints := make([]InterestingPoint, 0)

	class := c.Snapshot()
	file := class.File.Snapshot()

	interestingPoint := InterestingPoint{Text: class.Name, Kind: interfaces.SymbolKind.Class}
	interestingPoint.SetPosition(interfaces.OffsetNodeByNode(class.NameNode, class.Node))
	interestingPoint.SetFile(file.Content, file.URI)

	interestingPoints = append(interestingPoints, interestingPoint)

	for _, d := range c.Snapshot().Definitions {
		var locationNode *sitter.Node

		nameNode := d.Node.ChildByFieldName("name")
		if nameNode != nil {
			locationNode = nameNode
		} else {
			locationNode = d.Node
		}

		startOffset, endOffset := interfaces.OffsetNodeByNode(locationNode, class.Node)

		var kind interfaces.TSymbolKind

		nodeKind := d.Node.Type()
		if nodeKind == "method_definition" || nodeKind == "method_signature" || nodeKind == "abstract_method_signature" {
			kind = interfaces.SymbolKind.Method
		} else {
			kind = interfaces.SymbolKind.Property
		}

		interestingPoint := InterestingPoint{Text: class.Name + "." + d.Name, Kind: kind}
		interestingPoint.SetPosition(startOffset, endOffset)
		interestingPoint.SetFile(file.Content, file.URI)

		interestingPoints = append(interestingPoints, interestingPoint)
	}

	if c.HasComponent() && class.Angular.Component.SelectorNode != nil {
		startOffset := class.Angular.Component.SelectorNode.StartByte()
		endOffset := class.Angular.Component.SelectorNode.EndByte()

		for _, selector := range class.Angular.Component.Selectors {
			interestingPoint := InterestingPoint{Text: selector, Kind: interfaces.SymbolKind.Class}
			interestingPoint.SetPosition(startOffset, endOffset)
			interestingPoint.SetFile(file.Content, file.URI)

			interestingPoints = append(interestingPoints, interestingPoint)
		}
	}

	return interestingPoints
}

func (c *Class) GetSelectors() []string {
	if c.HasComponent() {
		return c.Snapshot().Angular.Component.Selectors
	}

	if c.HasDirective() {
		return c.Snapshot().Angular.Directive.Selectors
	}

	return []string{}
}

func (c *Class) FilterOwnDefinitions(cond func(d ClassedDefinition) bool) []ClassedDefinition {
	arr := []ClassedDefinition{}
	for _, definition := range c.GetClassedDefinitions() {
		if cond(definition) {
			arr = append(arr, definition)
		}
	}
	return arr
}

func (c *Class) FilterOwnDefinitionsOne(cond func(d ClassedDefinition) bool) *ClassedDefinition {
	for _, definition := range c.GetClassedDefinitions() {
		if cond(definition) {
			return &definition
		}
	}
	return nil
}

func (c *Class) FilterAllDefinitions(cond func(d ClassedDefinition) bool) []ClassedDefinition {
	definitions := c.FilterOwnDefinitions(cond)
	definitionsMap := make(map[string]bool)

	for _, d := range definitions {
		definitionsMap[d.Name] = true
	}

	for _, e := range c.Snapshot().Extends {
		if e == nil || e.Class == nil {
			continue
		}

		ds := e.Class.FilterAllDefinitions(cond)
		for _, d := range ds {
			// Don't allow duplicates. Also, prepare for doing stuff with overridden props
			found, _ := definitionsMap[d.Name]
			if !found {
				definitionsMap[d.Name] = true
				definitions = append(definitions, d)
			}
		}
	}

	return definitions
}

func (c *Class) FilterAllDefinitionsByDecorator(decoratorName string) []ClassedDefinition {
	return c.FilterAllDefinitions(func(def ClassedDefinition) bool {
		return slices.ContainsFunc(def.Decorators, func(dec Decorator) bool { return dec.Name == decoratorName })
	})
}

func (c *Class) FilterAllDefinitionsOne(cond func(d ClassedDefinition) bool) *ClassedDefinition {
	definition := c.FilterOwnDefinitionsOne(cond)
	if definition != nil {
		return definition
	}

	for _, e := range c.Snapshot().Extends {
		if e == nil || e.Class == nil {
			continue
		}

		ds := e.Class.FilterAllDefinitionsOne(cond)
		if ds != nil {
			return definition
		}
	}

	return nil
}

func (s *State) FindPlacesThatUseThisClassComponent(class *Class) []*Class {
	places := []*Class{}

	if !class.HasComponent() {
		return places
	}

	for _, c := range s.GetClasses() {
		if !c.HasComponent() {
			continue
		}

		template := c.Snapshot().Angular.Component.Template
		if template == nil {
			continue
		}

		for _, selector := range class.Snapshot().Angular.Component.Selectors {
			usages, found := template.TagUsages[selector]
			if !found || len(usages.Usages) == 0 {
				continue
			}

			places = append(places, c)
		}
	}

	return places
}

func (s *State) FindPlacesThatDeclareThisClassComponent(class *Class) []*Class {
	places := []*Class{}

	if !class.HasComponent() {
		return places
	}

	for _, c := range s.GetClasses() {
		if !c.HasModule() {
			continue
		}

		for declaration := range c.Snapshot().Angular.Module.Declarations.FlattenReferenceArraysToReferences(s) {
			declaration.Resolve(s)
			if declaration.Class == nil || declaration.Class != class {
				continue
			}

			places = append(places, c)
		}
	}

	return places
}

func (s *State) FindModulesThatAngularImportThisClass(class *Class) []*Class {
	places := []*Class{}

	for _, c := range s.GetClasses() {
		if !c.HasModule() {
			continue
		}

		if c.Snapshot().Angular.DoesImport(class) {
			places = append(places, c)
		}
	}

	return places
}

func (c *Class) GetAllProvidedValues(state *State) []*GetProvidersResult {
	visited := map[*Class]bool{}
	routesToRoot := [][]*Class{}

	rootTemplateUsages := state.FindPlacesThatUseThisClassComponent(c)
	for _, rootUsage := range rootTemplateUsages {
		routes := findProviderRoutes(state, []*Class{c}, visited, rootUsage)
		routesToRoot = append(routesToRoot, routes...)
	}

	rootModuleUsages := state.FindPlacesThatDeclareThisClassComponent(c)
	for _, rootUsage := range rootModuleUsages {
		routes := findProviderRoutes(state, []*Class{c}, visited, rootUsage)
		routesToRoot = append(routesToRoot, routes...)
	}

	providers := []*GetProvidersResult{}

	addProviderIfNotExists := func(treeProviders []*GetProvidersResult, providers []*Provider, class *Class) []*GetProvidersResult {
		for _, provider := range providers {
			if slices.ContainsFunc(treeProviders, func(p *GetProvidersResult) bool { return p.Provider.Token.Name == provider.Token.Name }) {
				continue
			}

			treeProviders = append(treeProviders, &GetProvidersResult{provider, class})
		}

		return treeProviders
	}

	for _, route := range routesToRoot {
		treeProviders := []*GetProvidersResult{}
		for _, entry := range route {
			if entry.HasComponent() {
				treeProviders = addProviderIfNotExists(treeProviders, entry.Snapshot().Angular.Component.Providers, entry)
			}

			if entry.HasModule() {
				treeProviders = addProviderIfNotExists(treeProviders, entry.Snapshot().Angular.Module.Providers, entry)
			}
		}

		providers = append(providers, treeProviders...)
	}

	return providers
}

func (c *Class) GetAllPublicDefinitions() []ClassedDefinition {
	definitions := c.GetOwnPublicDefinitions()
	definitionsMap := make(map[string]bool)

	for _, d := range definitions {
		definitionsMap[d.Name] = true
	}

	for _, e := range c.Snapshot().Extends {
		if e == nil || e.Class == nil {
			continue
		}

		ds := e.Class.GetAllPublicDefinitions()
		for _, d := range ds {
			// Don't allow duplicates. Also, prepare for doing stuff with overridden props
			found, _ := definitionsMap[d.Name]
			if !found {
				definitionsMap[d.Name] = true
				definitions = append(definitions, d)
			}
		}
	}

	return definitions
}

func (c *Class) GetAllDefinitions() []ClassedDefinition {
	return c.FilterAllDefinitions(func(d ClassedDefinition) bool { return true })
}

func (c *Class) GetClassedDefinitions() []ClassedDefinition {
	definitions := c.Snapshot().Definitions
	classedDefinitions := make([]ClassedDefinition, len(definitions))

	i := 0
	for _, d := range definitions {
		classedDefinitions[i] = ClassedDefinition{d, c}
		i++
	}

	return classedDefinitions
}

func (c *Class) GetDefinition(name string) *ClassedDefinition {
	return c.FilterAllDefinitionsOne(func(d ClassedDefinition) bool { return d.Name == name })
}

func (c *Class) GetDocumentation(includeClassName bool) string {
	documentation := make([]string, 0)

	if includeClassName {
		documentation = append(documentation, "# "+c.Snapshot().Name)
	}

	if c.HasComponent() && len(c.Snapshot().Angular.Component.DeclaredIn) > 0 {
		modules := make([]string, 0)

		for _, d := range c.Snapshot().Angular.Component.DeclaredIn {
			modules = append(modules, d.Snapshot().Name)
		}

		documentation = append(documentation, "**Declared in:** "+strings.Join(modules, ", "))
	}

	documentation = append(documentation, buildDefinitionSection("Inputs", c.GetInputs(true), true, false))
	documentation = append(documentation, buildDefinitionSection("Outputs", c.GetOutputs(), false, true))

	text := strings.Join(documentation, "\n\n")

	return text
}

func (c *Class) GetExtendsHierarchy() []*Class {
	classes := []*Class{}

	for e := range c.Snapshot().Extends.IterateResolved {
		classes = append(classes, e.Class)
		classes = append(classes, e.Class.GetExtendsHierarchy()...)
	}

	return classes
}

func (c *Class) GetExtendsImplementsHierarchy() []*Class {
	classes := c.GetExtendsHierarchy()
	classes = append(classes, c.GetImplementsHierarchy()...)

	return classes
}

func (c *Class) GetGetters() []ClassedDefinition {
	return c.FilterOwnDefinitions(func(d ClassedDefinition) bool { return d.Getter })
}

func (c *Class) GetInputs(sort bool) []ClassedDefinition {
	inputs := c.FilterAllDefinitionsByDecorator("Input")
	if sort {
		slices.SortFunc(inputs, func(a ClassedDefinition, b ClassedDefinition) int {
			return cmp.Compare(a.GetInputName(), b.GetInputName())
		})
	}

	return inputs
}

func (c *Class) GetImplementsHierarchy() []*Class {
	classes := []*Class{}

	for i := range c.Snapshot().Implements.IterateResolved {
		classes = append(classes, i.Class)
		classes = append(classes, i.Class.GetExtendsHierarchy()...)
	}

	return classes
}

func (c *Class) GetOutputs() []ClassedDefinition {
	return c.FilterAllDefinitionsByDecorator("Output")
}

func (c *Class) GetOwnDefinition(name string) *Definition {
	for _, d := range c.Snapshot().Definitions {
		if d.Name == name {
			return d
		}
	}

	return nil
}

func (c *Class) GetOwnPublicDefinitions() []ClassedDefinition {
	return c.FilterOwnDefinitions(func(d ClassedDefinition) bool { return d.IsPublic() })
}

func (c *Class) GetTemplateFile() *File {
	if c.Snapshot().Angular != nil && c.Snapshot().Angular.Component != nil {
		return c.Snapshot().Angular.Component.TemplateUrlFile
	}

	return nil
}

func (c *Class) HasAngular() bool {
	return c.Snapshot().Angular != nil
}

func (c *Class) HasComponent() bool {
	return c.HasAngular() && c.Snapshot().Angular.Component != nil
}

func (c *Class) HasDirective() bool {
	return c.HasAngular() && c.Snapshot().Angular.Directive != nil
}

func (c *Class) HasPipe() bool {
	return c.HasAngular() && c.Snapshot().Angular.Pipe != nil
}

func (c *Class) HasModule() bool {
	return c.HasAngular() && c.Snapshot().Angular.Module != nil
}

func (c *Class) Id() string { return ClassId(c.Snapshot().File.Snapshot().URI, c.Snapshot().Name) }

func (c *Class) Postprocess(state *State) {
	c.resolveExtendsImplements(state)

	c.removeOwnUagesUpwards()
	c.propagateOwnUagesUpwards()

	if c.Snapshot().Angular != nil {
		c.Snapshot().Angular.Postprocess(state, c)
	}
}

// Clears everything except the reference to the parent file
func (c *Class) Reset() {
	c.Update(func(data *classState) {
		data.Angular = nil
		data.Content = ""
		clear(data.Definitions)
		clear(data.Extends)
		data.ExtendsIdentNames = make([]string, 0)
		clear(data.Implements)
		data.ImplementsIdentNames = make([]string, 0)
		data.Name = ""
		data.Node = nil
		data.TypeParameters = make([]string, 0)
		clear(data.Usages)
	})
}

func (c *Class) ResetThings() {
	if c.HasComponent() {
		c.Snapshot().Angular.Component.ResetAvailableThings()
	}

	if c.HasModule() {
		c.Snapshot().Angular.Module.ResetExportedThings()
	}
}

func (c *Class) SetUsageAccessType(name string, access access) {
	usage := c.Snapshot().Usages[name]
	usage.Access = CalculateNewAccessType(access, usage.Access)
}

func (c *Class) Snapshot() classState {
	c.RLock()
	state := c.state
	c.RUnlock()

	return state
}

func (c *Class) Update(fn func(data *classState)) {
	c.Lock()
	defer c.Unlock()
	fn(&c.state)
}

func (c *Class) propagateOwnUagesUpwards() {
	usages := c.Snapshot().Usages
	c.propagateUsagesUpwards(usages)
}

func (c *Class) propagateUsagesUpwards(usages Usages) {
	for class := range c.Snapshot().Extends.IterateResolved {
		for name, usage := range usages {
			for _, instance := range usage.Usages {
				class.Class.AppendDefinitionUsage(name, instance)
			}
		}

		class.Class.propagateUsagesUpwards(usages)
	}
}

func (c *Class) removeOwnUagesUpwards() {
	c.removeUsagesFromClassUpwards(c)
}

// Remove all of the usages originating from a particular class all the way up the hierarchy
func (c *Class) removeUsagesFromClassUpwards(class *Class) {
	for e := range c.Snapshot().Extends.IterateResolved {
		definitions := e.Class.Snapshot().Definitions
		for name, d := range definitions {
			newUsageInstances := []*UsageInstance{}

			for _, instance := range d.Usages {
				if instance.Class == class {
					continue
				}

				newUsageInstances = append(newUsageInstances, instance)
				d.UsageAccess = CalculateNewAccessType(d.UsageAccess, instance.Access)
			}

			definition := definitions[name]
			definition.Usages = newUsageInstances
			definitions[name] = definition
		}

		e.Class.Update(func(data *classState) {
			data.Definitions = definitions
		})

		e.Class.removeUsagesFromClassUpwards(class)
	}
}

func (c *Class) resolveExtendsImplements(state *State) {
	file := c.Snapshot().File
	extends := resolveIdents(c.Snapshot().ExtendsIdentNames, file, state)
	implements := resolveIdents(c.Snapshot().ImplementsIdentNames, file, state)

	c.Update(func(data *classState) {
		data.Extends = extends
		data.Implements = implements
	})
}

func ClassId(uri string, className string) string { return uri + "-" + className }

func NewClass(content string, file *File, node *sitter.Node) Class {
	state := classState{
		Content:              content,
		Definitions:          make(map[string]*Definition),
		Extends:              []*Reference{},
		ExtendsIdentNames:    []string{},
		File:                 file,
		Implements:           []*Reference{},
		ImplementsIdentNames: []string{},
		Name:                 "",
		Node:                 node,
		TypeParameters:       []string{},
		Usages:               make(map[string]Usage),
	}

	return Class{state: state}
}

func buildDefinitionSection(sectionName string, definitions []ClassedDefinition, isInput bool, isOutput bool) string {
	section := make([]string, 0)

	slices.SortFunc(definitions, func(a ClassedDefinition, b ClassedDefinition) int { return cmp.Compare(a.Name, b.Name) })
	if len(definitions) > 0 {
		section = append(section, "**"+sectionName+":**")
		for _, def := range definitions {
			var name string
			if isInput {
				name = def.GetInputName()
			} else if isOutput {
				name = def.GetOutputName()
			}

			section = append(section, "  "+name+" (*"+def.Class.Snapshot().Name+"*)")
		}
	}

	return strings.Join(section, "\n")
}

func findProviderRoutes(state *State, path []*Class, visited map[*Class]bool, class *Class) [][]*Class {
	routesToTarget := [][]*Class{}

	if value, found := visited[class]; found && value {
		return routesToTarget
	}

	visited[class] = true

	if !class.HasComponent() && !class.HasModule() {
		return routesToTarget
	}

	if class.HasComponent() {
		routesToTarget = append(routesToTarget, findProviderRoutesComponent(state, path, visited, class)...)
	}

	if class.HasModule() {
		routesToTarget = append(routesToTarget, findProviderRoutesModule(state, path, visited, class)...)
	}

	return routesToTarget
}

func findProviderRoutesModule(state *State, path []*Class, visited map[*Class]bool, class *Class) [][]*Class {
	routesToTarget := [][]*Class{}

	importingClasses := state.FindModulesThatAngularImportThisClass(class)
	for _, importingClass := range importingClasses {
		foundRoutes := findProviderRoutes(state, append(path, class), visited, importingClass)
		routesToTarget = append(routesToTarget, foundRoutes...)
	}

	if len(routesToTarget) == 0 {
		routesToTarget = append(routesToTarget, append(path, class))
	}

	return routesToTarget
}

func findProviderRoutesComponent(state *State, path []*Class, visited map[*Class]bool, class *Class) [][]*Class {
	routesToTarget := [][]*Class{}

	tagUsages := state.FindPlacesThatUseThisClassComponent(class)

	// It's a root terminal component
	if class.HasComponent() && len(tagUsages) == 0 {
		routesToTarget = append(routesToTarget, append(path, class))

		return routesToTarget
	}

	for _, tagUsage := range tagUsages {
		foundRoutes := findProviderRoutes(state, append(path, class), visited, tagUsage)
		routesToTarget = append(routesToTarget, foundRoutes...)
	}

	moduleDeclarations := state.FindPlacesThatDeclareThisClassComponent(class)
	for _, moduleDeclaration := range moduleDeclarations {
		routes := findProviderRoutes(state, append(path, class), visited, moduleDeclaration)
		routesToTarget = append(routesToTarget, routes...)
	}

	return routesToTarget
}
