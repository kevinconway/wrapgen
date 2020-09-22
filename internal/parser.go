package wrapgen

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

func loadPackage(ctx context.Context, path string) (*packages.Package, error) {
	fset := token.NewFileSet()
	conf := &packages.Config{
		Mode:    packages.LoadAllSyntax,
		Context: ctx,
		Fset:    fset,
	}
	pkgs, err := packages.Load(conf, path)
	if err != nil {
		return nil, err
	}
	if len(pkgs) < 1 {
		return nil, fmt.Errorf("%s not found\n", path)
	}
	if len(pkgs) > 2 {
		return nil, fmt.Errorf(
			"%s contains too many packages. expected at most 2 (pkg, pkg_test). found: %s\n",
			path, pkgs,
		)
	}
	var pkgErrors []error
	for _, pkg := range pkgs {
		for _, pkgErr := range pkg.Errors {
			// []packages.Error cannot conver to []error even though all the
			// contained types are valid error implementations. Because of this
			// we must individually iterate and append rather than using append
			// with the variadic notation (ex: append(x, y...)).
			pkgErrors = append(pkgErrors, pkgErr)
		}
	}
	if len(pkgErrors) > 0 {
		return nil, multiError(pkgErrors)
	}
	var pkg *packages.Package
	for _, p := range pkgs {
		if strings.Contains(p.Name, "_test") {
			continue
		}
		pkg = p
	}
	if pkg == nil {
		return nil, fmt.Errorf("%s only contains test packages", path)
	}
	return pkg, nil
}

func LoadInterfaces(ctx context.Context, srcPkg, srcPkgAlias string, names []string) ([]*Import, []*Interface, error) {
	pkg, err := loadPackage(ctx, srcPkg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load package data: %v", err)
	}
	var (
		imports []*Import
		ifaces  []*Interface
	)
	for _, name := range names {
		imps, iface, err := loadInterface(ctx, pkg, name, srcPkgAlias)
		if err != nil {
			return nil, nil, err
		}
		imports = append(imports, imps...)
		ifaces = append(ifaces, iface)
	}
	return imports, ifaces, nil
}

func loadInterface(ctx context.Context, pkg *packages.Package, name, srcPkgAlias string) ([]*Import, *Interface, error) {
	for _, f := range pkg.Syntax {
		localImport := locals(ctx, pkg, f)
		if srcPkgAlias != "" {
			localImport[pkg.PkgPath] = srcPkgAlias
		}
		for _, decl := range f.Decls {
			switch dd := decl.(type) {
			case *ast.GenDecl:
				// https://golang.org/pkg/go/ast/#GenDecl
				// For ease of understanding, here are some details from the
				// official documentation linked above:
				//
				// A GenDecl node (generic declaration node) represents an import,
				// constant, type or variable declaration.
				// Relationship between Tok value and Specs element type:
				// token.IMPORT  *ImportSpec
				// token.CONST   *ValueSpec
				// token.TYPE    *TypeSpec
				// token.VAR     *ValueSpec
				//
				// Because of the relationship between the token type and the
				// underlying spec value, we are able to perform a comparison check
				// here which prevents the need to do more type switching on the
				// spec value.
				if dd.Tok != token.TYPE {
					// Full list of token types can be found here:
					// https://golang.org/pkg/go/token/#Token.
					continue
				}
				for _, spec := range dd.Specs {
					ss, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if ss.Name == nil || ss.Name.Name != name {
						continue
					}
					// https://golang.org/pkg/go/ast/#TypeSpec
					// TypeSpec instances represent anywhere a new type is defined.
					// TypeSpecs are categorized by their own SrcType field which
					// indicates the kind of type definition. The possible values
					// from the docs are:
					//
					// - *Ident
					// - *ParenExpr
					// - *SelectorExpr
					// - *StarExpr
					// - any of the *XxxTypes
					//
					// The "any of the *XxxTypes" refers to other concrete elements
					// from the ast modules such as ArrayType or ChanType. Since
					// this project is scoped only to handling interface types we
					// can safely ignore all of the concrete types as they are not
					// relevant. The *StarExpr is also ignored because cases of a
					// a pointer to an interface (ex: `type T *io.Writer`) don't
					// really make sense as the pointer has no methods to capture.
					//
					// Each relevant case is captured and documented in the switch
					// below.
					switch ff := ss.Type.(type) {
					case *ast.InterfaceType:
						// InterfaceType is an easy case where something has been
						// defined as `type T interface{}`. This is the clearest
						// and easiest to handle case.
						return parseInterface(ctx, pkg, localImport, ss.Name.String(), srcPkgAlias, ff)
					case *ast.ParenExpr:
						// It's not clear from the docs exactly how a ParenExpr
						// might appear as a TypeSpec category. Placing this error
						// here in the hopes that whoever finds this case will open
						// an issue so we can implement it correctly using their
						// use case as a guide.
						return nil, nil, fmt.Errorf(
							"unhandled ParenExpr for type %s. please report this issue",
							ss.Name,
						)
					case *ast.Ident:
						// https://golang.org/pkg/go/ast/#Ident
						// Ident covers cases of aliases and subtypes where
						// there is no use of the `.` character on the right-hand
						// side.
						if ff.Obj == nil {
							// According to the docs, the Ident ast node has an optional
							// `Obj` field which may be either `nil` or a reference to
							// the entity referenced. This presents a challenge for
							// cases where the `Obj` is `nil` because we have no
							// reliable way of determining how to get the data we need.
							// For now, we will emit an error for this case and decide
							// later if we can support it or not.
							return nil, nil, fmt.Errorf(
								"unhandled missing Obj in Ident for type %s. please report this issue",
								ss.Name,
							)
						}
						if ff.Obj.Kind != ast.Typ {
							// https://golang.org/pkg/go/ast/#ObjKind
							// The only object kind we support is Typ as that is
							// the only kind that may be an interface.
							continue
						}
						// For all cases found so far, an Obj kind of Typ contains
						// a non-nil Decl reference to either the TypeSpec of
						// the type being aliased/extended or the SelectorExpr
						// of a remote type being reference..
						switch fft := ff.Obj.Decl.(*ast.TypeSpec).Type.(type) {
						case *ast.InterfaceType:
							return parseInterface(ctx, pkg, localImport, ss.Name.String(), srcPkgAlias, fft)
						case *ast.SelectorExpr:
							// This is a curious case where the right-hand side
							// may actually resolve to a SelectorExpr when the
							// value is the name of a local alias that _was_
							// defined with a SelectorExpr. For example:
							//
							// type T io.Writer
							// type T2 T
							//
							// The handling logic is exactly the same as if the
							// SelectorExpr case of the TypeSpec switch. The
							// explanation is documented in more detail there.
							if _, ok := fft.X.(*ast.Ident); !ok {
								return nil, nil, fmt.Errorf(
									"cannot interpret %s in %s. expression too complex",
									name, pkg.PkgPath,
								)
							}
							remotePkgName := fft.X.(*ast.Ident).Name
							remoteItemName := fft.Sel.Name
							remotePkgName = localImport[remotePkgName]
							remotePkg := pkg.Imports[remotePkgName]
							var u, ifs, err = loadInterface(ctx, remotePkg, remoteItemName, srcPkgAlias)
							if err != nil {
								return nil, nil, err
							}
							ifs.Name = name
							return append(u, &Import{Path: remotePkg.PkgPath, Package: remotePkg.Name}), ifs, nil
						default:
							fmt.Printf("%T\n", fft)
							return nil, nil, fmt.Errorf(
								"%s in %s is not an interface", name, pkg.PkgPath,
							)
						}
					case *ast.SelectorExpr:
						// https://golang.org/pkg/go/ast/#SelectorExpr
						// SelectorExpr represents a case of defining an alias or
						// subtype for a type from a different package. This node
						// has two attributes, X and Sel. The X attribute can be
						// any kind of expression and the Sel attribute is an
						// Identity node that represents what comes after the `.`
						// character. For example, some valid selector expressions
						// might be:
						//
						// - t.(T).Attribute
						// - t.Method
						// - pkg.T
						//
						// Each of these might resolve to a different expression
						// type for X. In the context of type definitions, the only
						// case we cover is when X is an Identity node denoting the
						// package name (ex: `type T pkg.T`).
						if _, ok := ff.X.(*ast.Ident); !ok {
							return nil, nil, fmt.Errorf(
								"cannot interpret %s in %s. expression too complex",
								name, pkg.PkgPath,
							)
						}
						remotePkgName := ff.X.(*ast.Ident).Name
						remoteItemName := ff.Sel.Name
						// Unfortunately, unlike the Ident case in this switch, we
						// are always given `nil` for both the X and Sel values of
						// Obj which means we are now responsible for finding the
						// relevant content. This is the reason we need access to
						// the global map of package data. To do this we must first
						// convert the local selector name into a full package path.
						// We do this because the remote package may be aliased as
						// any arbitrary name within the current file. Other files
						// in the same package may also use that same alias name but
						// for a different imported package. The result is that we
						// have to track the file-local names of imports and convert
						// them where we encounter them.
						remotePkgName = localImport[remotePkgName]
						remotePkg := pkg.Imports[remotePkgName]
						var u, ifs, err = loadInterface(ctx, remotePkg, remoteItemName, srcPkgAlias)
						if err != nil {
							return nil, nil, err
						}
						ifs.Name = name
						return append(u, &Import{Path: remotePkg.PkgPath, Package: remotePkg.Name}), ifs, nil
					default:
						return nil, nil, fmt.Errorf(
							"%s in %s is not an interface.", name, pkg.PkgPath,
						)
					}
				}
			default:
				// Since all type definitions match GenDecl we can safely ignore
				// everything else.
				continue
			}
		}
	}
	return nil, nil, fmt.Errorf("interface %s not found in package %s", name, pkg.PkgPath)
}

func parseType(ctx context.Context, pkg *packages.Package, locals map[string]string, arg ast.Expr) ([]*Import, Type, error) {
	switch n := arg.(type) {
	case *ast.ArrayType:
		var len = -1
		if n.Len != nil {
			len, _ = strconv.Atoi(n.Len.(*ast.BasicLit).Value)
		}
		var u, typ, e = parseType(ctx, pkg, locals, n.Elt)
		if e != nil {
			return nil, nil, e
		}
		return u, &TypeArray{Len: len, Type: typ}, nil
	case *ast.ChanType:
		var u, t, e = parseType(ctx, pkg, locals, n.Value)
		if e != nil {
			return nil, nil, e
		}
		var chanType = &TypeChan{Type: t}
		if n.Dir == ast.SEND {
			chanType.WriteOnly = true
		}
		if n.Dir == ast.RECV {
			chanType.ReadOnly = true
		}
		return u, chanType, nil
	case *ast.Ellipsis:
		var u, t, e = parseType(ctx, pkg, locals, n.Elt)
		if e != nil {
			return nil, nil, e
		}
		return u, &TypeVariadic{Type: t}, nil
	case *ast.FuncType:
		var u, method, e = parseFunc(ctx, pkg, locals, "", n)
		if e != nil {
			return nil, nil, e
		}
		var result = &TypeFunc{In: make([]Type, 0), Out: make([]Type, 0)}
		for _, param := range method.In {
			result.In = append(result.In, param.Type)
		}
		for _, param := range method.Out {
			result.Out = append(result.Out, param.Type)
		}
		return u, result, nil
	case *ast.Ident:
		if n.IsExported() {
			// alias indicate we want to alias a type in this package
			if alias := locals[pkg.PkgPath]; alias != "" {
				return []*Import{{Path: pkg.PkgPath, Package: alias}}, &TypeExported{Package: alias, Type: TypeBuiltin(n.Name)}, nil
			}
			// assume type in this package
			return nil, TypeBuiltin(n.Name), nil
		}
		return nil, TypeBuiltin(n.Name), nil
	case *ast.InterfaceType:
		if n.Methods != nil && len(n.Methods.List) > 0 {
			return nil, nil, fmt.Errorf("can't handle non-empty unnamed interface types at %v", n.Pos())
		}
		return nil, TypeBuiltin("interface{}"), nil
	case *ast.MapType:
		var key Type
		var uKey []*Import
		var value Type
		var uValue []*Import
		var e error
		uKey, key, e = parseType(ctx, pkg, locals, n.Key)
		if e != nil {
			return nil, nil, e
		}
		uValue, value, e = parseType(ctx, pkg, locals, n.Value)
		if e != nil {
			return nil, nil, e
		}
		return append(uKey, uValue...), &TypeMap{Key: key, Value: value}, nil
	case *ast.SelectorExpr:
		pkgName := n.X.(*ast.Ident).String()
		pkgName = locals[pkgName]
		remotePkg := pkg.Imports[pkgName]
		return []*Import{{Path: remotePkg.PkgPath, Package: remotePkg.Name}}, &TypeExported{Package: remotePkg.Name, Type: TypeBuiltin(n.Sel.String())}, nil
	case *ast.StarExpr:
		var u, t, e = parseType(ctx, pkg, locals, n.X)
		if e != nil {
			return nil, nil, e
		}
		return u, &TypePointer{Type: t}, nil
	case *ast.StructType:
		if n.Fields != nil && len(n.Fields.List) > 0 {
			return nil, nil, fmt.Errorf("can't handle non-empty unnamed struct types at %v", n.Pos())
		}
		return nil, TypeBuiltin("struct{}"), nil
	}
	return nil, nil, fmt.Errorf("unknown type: %T", arg)
}

func parseFunc(ctx context.Context, pkg *packages.Package, locals map[string]string, name string, f *ast.FuncType) ([]*Import, *Method, error) {
	var method = &Method{Name: name, In: make([]*Parameter, 0), Out: make([]*Parameter, 0)}
	var used []*Import
	if f.Params != nil {
		for offset, arg := range f.Params.List {
			var param = &Parameter{Name: fmt.Sprintf("param%d", offset)}
			if len(arg.Names) > 0 {
				param.Name = arg.Names[0].String()
			}
			var u, t, e = parseType(ctx, pkg, locals, arg.Type)
			if e != nil {
				return nil, nil, e
			}
			used = append(used, u...)
			param.Type = t
			method.In = append(method.In, param)
		}
	}
	if f.Results != nil {
		for offset, arg := range f.Results.List {
			var param = &Parameter{Name: fmt.Sprintf("result%d", offset)}
			if len(arg.Names) > 0 {
				param.Name = arg.Names[0].String()
			}
			var u, t, e = parseType(ctx, pkg, locals, arg.Type)
			if e != nil {
				return nil, nil, e
			}
			used = append(used, u...)
			param.Type = t
			method.Out = append(method.Out, param)
		}
	}
	return used, method, nil
}

func parseInterface(ctx context.Context, pkg *packages.Package, locals map[string]string, name, srcPkgAlias string, i *ast.InterfaceType) ([]*Import, *Interface, error) {
	var ifcType Type
	if alias := locals[pkg.PkgPath]; alias != "" {
		ifcType = &TypeExported{Package: alias, Type: TypeBuiltin(name)}
	} else {
		ifcType = TypeBuiltin(name)
	}
	var iface = &Interface{SrcType: ifcType, Name: name, Methods: make([]*Method, 0)}
	var used []*Import
	for _, attribute := range i.Methods.List {
		switch n := attribute.Type.(type) {
		case *ast.FuncType:
			var u, m, e = parseFunc(ctx, pkg, locals, attribute.Names[0].String(), n)
			if e != nil {
				return nil, nil, e
			}
			used = append(used, u...)
			iface.Methods = append(iface.Methods, m)
		case *ast.Ident:
			var u, ifs, err = loadInterface(ctx, pkg, n.String(), srcPkgAlias)
			if err != nil {
				return nil, nil, fmt.Errorf(
					"missing local embedded interface %s: %v",
					n.String(), err,
				)
			}
			used = append(used, u...)
			iface.Methods = append(iface.Methods, ifs.Methods...)
		case *ast.SelectorExpr:
			pkgName := n.X.(*ast.Ident).String()
			pkgName = locals[pkgName]
			remotePkg := pkg.Imports[pkgName]
			u, ifs, err := loadInterface(ctx, remotePkg, n.Sel.String(), srcPkgAlias)
			if err != nil {
				return nil, nil, fmt.Errorf(
					"missing remote embedded interface %s.%s: %v",
					remotePkg.PkgPath, n.Sel.String(), err,
				)
			}
			used = append(used, u...)
			used = append(used, &Import{Path: remotePkg.PkgPath, Package: remotePkg.Name})
			iface.Methods = append(iface.Methods, ifs.Methods...)
		default:
			continue
		}
	}
	return used, iface, nil
}

func locals(ctx context.Context, pkg *packages.Package, f *ast.File) map[string]string {
	results := make(map[string]string)
	for _, imp := range f.Imports {
		pth := imp.Path.Value[1 : len(imp.Path.Value)-1]
		if imp.Name != nil {
			results[imp.Name.Name] = pth
			continue
		}
		results[pkg.Imports[pth].Name] = pth
	}
	return results
}
