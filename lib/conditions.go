package lib

import (
	http_cel "codeberg.org/gone/http-cel"
	"fmt"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"log/slog"
	"net"
	"strings"
)

func (state *State) initConditions() (err error) {
	state.programEnv, err = http_cel.NewEnvironment(

		cel.Variable("fp", cel.MapType(cel.StringType, cel.StringType)),
		cel.Function("inDNSBL",
			cel.Overload("inDNSBL_ip",
				[]*cel.Type{cel.AnyType},
				cel.BoolType,
				cel.UnaryBinding(func(val ref.Val) ref.Val {
					slog.Error("inDNSBL function has been deprecated, replace with dnsbl challenge")
					return types.Bool(false)
				}),
			),
		),

		cel.Function("network",
			cel.MemberOverload("netIP_network_string",
				[]*cel.Type{cel.BytesType, cel.StringType},
				cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					var ip net.IP
					switch v := lhs.Value().(type) {
					case []byte:
						ip = v
					case net.IP:
						ip = v
					}

					if ip == nil {
						panic(fmt.Errorf("invalid ip %v", lhs.Value()))
					}

					val, ok := rhs.Value().(string)
					if !ok {
						panic(fmt.Errorf("invalid network value %v", rhs.Value()))
					}

					network, ok := state.networks[val]
					if !ok {
						_, ipNet, err := net.ParseCIDR(val)
						if err != nil {
							panic("network not found")
						}
						return types.Bool(ipNet.Contains(ip))
					} else {
						ok, err := network().Contains(ip)
						if err != nil {
							panic(err)
						}
						return types.Bool(ok)
					}
				}),
			),
		),

		cel.Function("country",
			cel.MemberOverload("netIP_country_string",
				[]*cel.Type{cel.BytesType, cel.StringType},
				cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					var ip net.IP
					switch v := lhs.Value().(type) {
					case []byte:
						ip = v
					case net.IP:
						ip = v
					default:
						panic(fmt.Errorf("invalid ip type %T, expected []byte or net.IP", lhs.Value()))
					}

					if ip == nil {
						panic(fmt.Errorf("invalid ip %v", lhs.Value()))
					}

					countryValue, ok := rhs.Value().(string)
					if !ok {
						panic(fmt.Errorf("invalid country value %v, expected string", rhs.Value()))
					}

					// Check if we have a GeoIP database configured
					if state.geoipDB == nil {
						panic(fmt.Errorf("geoip database not configured"))
					}

					ipStr := ip.String()
					
					// Check cache first
					var countryCode, countryName string
					if cached, exists := state.geoipCache[ipStr]; exists {
						countryCode = cached.IsoCode
						countryName = cached.Name
					} else {
						// Lookup in database
						record, err := state.geoipDB.Country(ip)
						if err != nil {
							// Cache negative results too
							if state.geoipCache == nil {
								state.geoipCache = make(map[string]GeoIPCacheEntry)
							}
							state.geoipCache[ipStr] = GeoIPCacheEntry{IsoCode: "", Name: ""}
							return types.Bool(false)
						}
						
						countryCode = record.Country.IsoCode
						countryName = record.Country.Names["en"]
						
						// Add to cache (with simple size limit)
						if state.geoipCache == nil {
							state.geoipCache = make(map[string]GeoIPCacheEntry)
						}
						if len(state.geoipCache) < state.maxGeoIPCacheSize {
							state.geoipCache[ipStr] = GeoIPCacheEntry{
								IsoCode: countryCode,
								Name:    countryName,
							}
						}
					}

					// If we cached a negative result
					if countryCode == "" && countryName == "" {
						return types.Bool(false)
					}

					// Normalize input for comparison
					inputCountry := strings.ToUpper(strings.TrimSpace(countryValue))
					
					// Check against ISO country code (2-letter, e.g., "US", "GB")
					if strings.ToUpper(countryCode) == inputCountry {
						return types.Bool(true)
					}

					// Check against country name (e.g., "United States", "United Kingdom")
					if strings.ToUpper(countryName) == inputCountry {
						return types.Bool(true)
					}

					return types.Bool(false)
				}),
			),
			cel.MemberOverload("netIP_country_list",
				[]*cel.Type{cel.BytesType, cel.ListType(cel.StringType)},
				cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					var ip net.IP
					switch v := lhs.Value().(type) {
					case []byte:
						ip = v
					case net.IP:
						ip = v
					default:
						panic(fmt.Errorf("invalid ip type %T, expected []byte or net.IP", lhs.Value()))
					}

					if ip == nil {
						panic(fmt.Errorf("invalid ip %v", lhs.Value()))
					}

					countryList, ok := rhs.Value().([]ref.Val)
					if !ok {
						panic(fmt.Errorf("invalid country list %v, expected []string", rhs.Value()))
					}

					// Check if we have a GeoIP database configured
					if state.geoipDB == nil {
						panic(fmt.Errorf("geoip database not configured"))
					}

					ipStr := ip.String()
					
					// Check cache first
					var countryCode, countryName string
					if cached, exists := state.geoipCache[ipStr]; exists {
						countryCode = cached.IsoCode
						countryName = cached.Name
					} else {
						// Lookup in database
						record, err := state.geoipDB.Country(ip)
						if err != nil {
							// Cache negative results too
							if state.geoipCache == nil {
								state.geoipCache = make(map[string]GeoIPCacheEntry)
							}
							state.geoipCache[ipStr] = GeoIPCacheEntry{IsoCode: "", Name: ""}
							return types.Bool(false)
						}
						
						countryCode = record.Country.IsoCode
						countryName = record.Country.Names["en"]
						
						// Add to cache (with simple size limit)
						if state.geoipCache == nil {
							state.geoipCache = make(map[string]GeoIPCacheEntry)
						}
						if len(state.geoipCache) < state.maxGeoIPCacheSize {
							state.geoipCache[ipStr] = GeoIPCacheEntry{
								IsoCode: countryCode,
								Name:    countryName,
							}
						}
					}

					// If we cached a negative result
					if countryCode == "" && countryName == "" {
						return types.Bool(false)
					}

					actualCountryCode := strings.ToUpper(countryCode)
					actualCountryName := strings.ToUpper(countryName)

					// Check if IP's country matches any in the list
					for _, countryVal := range countryList {
						countryStr, ok := countryVal.Value().(string)
						if !ok {
							continue
						}
						
						inputCountry := strings.ToUpper(strings.TrimSpace(countryStr))
						
						if actualCountryCode == inputCountry || actualCountryName == inputCountry {
							return types.Bool(true)
						}
					}

					return types.Bool(false)
				}),
			),
		),

		cel.Function("inNetwork",
			cel.Overload("inNetwork_string_ip",
				[]*cel.Type{cel.StringType, cel.BytesType},
				cel.BoolType,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					var ip net.IP
					switch v := rhs.Value().(type) {
					case []byte:
						ip = v
					case net.IP:
						ip = v
					}

					if ip == nil {
						panic(fmt.Errorf("invalid ip %v", rhs.Value()))
					}

					val, ok := lhs.Value().(string)
					if !ok {
						panic(fmt.Errorf("invalid value %v", lhs.Value()))
					}
					slog.Debug(fmt.Sprintf("inNetwork function has been deprecated and will be removed in a future release, use remoteAddress.network(\"%s\") instead", val))

					network, ok := state.networks[val]
					if !ok {
						_, ipNet, err := net.ParseCIDR(val)
						if err != nil {
							panic("network not found")
						}
						return types.Bool(ipNet.Contains(ip))
					} else {
						ok, err := network().Contains(ip)
						if err != nil {
							panic(err)
						}
						return types.Bool(ok)
					}
				}),
			),
		),
	)
	if err != nil {
		return err
	}
	return nil
}

func (state *State) RegisterCondition(operator string, conditions ...string) (cel.Program, error) {
	compiledAst, err := http_cel.NewAst(state.ProgramEnv(), operator, conditions...)
	if err != nil {
		return nil, err
	}

	if out := compiledAst.OutputType(); out == nil {
		return nil, fmt.Errorf("no output")
	} else if out != types.BoolType {
		return nil, fmt.Errorf("output type is not bool")
	}

	walkExpr(compiledAst.NativeRep().Expr(), func(e ast.Expr) {
		if e.Kind() == ast.CallKind {
			call := e.AsCall()
			switch call.FunctionName() {
			// deprecated
			case "inNetwork":
				args := call.Args()
				if !call.IsMemberFunction() && len(args) == 2 {
					// we have a network select function
					switch args[1].Kind() {
					case ast.LiteralKind:
						lit := args[1].AsLiteral()
						if lit.Type() == types.StringType {
							if fn, ok := state.networks[lit.Value().(string)]; ok {
								// preload
								fn()
							}
						}
					}

				}
			case "network":
				args := call.Args()
				if call.IsMemberFunction() && len(args) == 1 {
					// we have a network select function
					switch args[0].Kind() {
					case ast.LiteralKind:
						lit := args[0].AsLiteral()
						if lit.Type() == types.StringType {
							if fn, ok := state.networks[lit.Value().(string)]; ok {
								// preload
								fn()
							}
						}
					}

				}
			}
		}
	})

	return http_cel.ProgramAst(state.ProgramEnv(), compiledAst)
}

func walkExpr(e ast.Expr, fn func(ast.Expr)) {
	fn(e)

	switch e.Kind() {
	case ast.CallKind:
		ee := e.AsCall()
		walkExpr(ee.Target(), fn)
		for _, arg := range ee.Args() {
			walkExpr(arg, fn)
		}
	case ast.ComprehensionKind:
		ee := e.AsComprehension()
		walkExpr(ee.Result(), fn)
		walkExpr(ee.IterRange(), fn)
		walkExpr(ee.AccuInit(), fn)
		walkExpr(ee.LoopCondition(), fn)
		walkExpr(ee.LoopStep(), fn)
	case ast.ListKind:
		ee := e.AsList()
		for _, element := range ee.Elements() {
			walkExpr(element, fn)
		}
	case ast.MapKind:
		ee := e.AsMap()
		for _, entry := range ee.Entries() {
			switch entry.Kind() {
			case ast.MapEntryKind:
				eee := entry.AsMapEntry()
				walkExpr(eee.Key(), fn)
				walkExpr(eee.Value(), fn)
			case ast.StructFieldKind:
				eee := entry.AsStructField()
				walkExpr(eee.Value(), fn)
			}
		}
	case ast.SelectKind:
		ee := e.AsSelect()
		walkExpr(ee.Operand(), fn)
	case ast.StructKind:
		ee := e.AsStruct()
		for _, field := range ee.Fields() {
			switch field.Kind() {
			case ast.MapEntryKind:
				eee := field.AsMapEntry()
				walkExpr(eee.Key(), fn)
				walkExpr(eee.Value(), fn)
			case ast.StructFieldKind:
				eee := field.AsStructField()
				walkExpr(eee.Value(), fn)
			}
		}
	}
}
