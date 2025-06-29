package server

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/L3m0nSo/Memories/server/actionresponse"
	"github.com/L3m0nSo/Memories/server/auth"
	daptinid "github.com/L3m0nSo/Memories/server/id"
	"github.com/L3m0nSo/Memories/server/resource"
	"github.com/L3m0nSo/Memories/server/table_info"
	"github.com/artpar/api2go/v2"
	"github.com/gobuffalo/flect"
	"github.com/google/uuid"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/relay"
	"github.com/iancoleman/strcase"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	//	"encoding/base64"
	"errors"
	//"fmt"
	"fmt"

	"github.com/artpar/api2go/v2/jsonapi"
)

var nodeDefinitions *relay.NodeDefinitions

var Schema graphql.Schema

func MakeGraphqlSchema(cmsConfig *resource.CmsConfig, resources map[string]*resource.DbResource) *graphql.Schema {

	//mutations := make(graphql.InputObjectConfigFieldMap)
	//query := make(graphql.InputObjectConfigFieldMap)
	//done := make(map[string]bool)

	inputTypesMap := make(map[string]*graphql.Object)
	//outputTypesMap := make(map[string]graphql.Output)
	//connectionMap := make(map[string]*relay.GraphQLConnectionDefinitions)

	nodeDefinitions = relay.NewNodeDefinitions(relay.NodeDefinitionsConfig{
		IDFetcher: func(id string, info graphql.ResolveInfo, ctx context.Context) (interface{}, error) {
			resolvedID := relay.FromGlobalID(id)

			ur, _ := url.Parse("/api/" + resolvedID.Type)
			pr := &http.Request{
				Method: "GET",
				URL:    ur,
			}
			pr = pr.WithContext(ctx)
			req := api2go.Request{
				PlainRequest: pr,
			}
			responder, err := resources[strings.ToLower(resolvedID.Type)].FindOne(resolvedID.ID, req)
			if responder != nil && responder.Result() != nil {
				return responder.Result().(api2go.Api2GoModel).GetAttributes(), err
			}
			return nil, err
		},
		TypeResolve: func(p graphql.ResolveTypeParams) *graphql.Object {
			log.Printf("Type resolve query: %v", p)
			//return inputTypesMap[p.Value]
			return nil
		},
	})

	rootFields := make(graphql.Fields)
	mutationFields := make(graphql.Fields)

	actionResponseType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "ActionResponse",
		Description: "Action response",
		Fields: graphql.Fields{
			"ResponseType": &graphql.Field{
				Type: graphql.String,
			},
			"Attributes": &graphql.Field{
				Type: graphql.NewObject(graphql.ObjectConfig{
					Name: "Attributes",
					Fields: graphql.Fields{
						"type": &graphql.Field{
							Name: "type",
							Type: graphql.String,
						},
						"message": &graphql.Field{
							Name: "message",
							Type: graphql.String,
						},
						"key": &graphql.Field{
							Name: "key",
							Type: graphql.String,
						},
						"value": &graphql.Field{
							Name: "value",
							Type: graphql.String,
						},
						"token": &graphql.Field{
							Name: "token",
							Type: graphql.String,
						},
						"title": &graphql.Field{
							Name: "title",
							Type: graphql.String,
						},
						"delay": &graphql.Field{
							Name: "delay",
							Type: graphql.String,
						},
						"location": &graphql.Field{
							Name: "location",
							Type: graphql.String,
						},
					},
				}),
			},
		},
	})

	pageConfig := graphql.ArgumentConfig{
		Type: graphql.NewInputObject(graphql.InputObjectConfig{
			Name:        "page",
			Description: "Page size and number",
			Fields: graphql.InputObjectConfigFieldMap{
				"number": &graphql.InputObjectFieldConfig{
					Type:         graphql.Int,
					DefaultValue: 1,
					Description:  "page number to fetch",
				},
				"size": &graphql.InputObjectFieldConfig{
					Type:         graphql.Int,
					DefaultValue: 10,
					Description:  "number of records in one page",
				},
			},
		}),
		Description:  "filter results by search query",
		DefaultValue: "",
	}

	queryArgument := graphql.ArgumentConfig{
		Type: graphql.NewList(graphql.NewInputObject(graphql.InputObjectConfig{
			Name:        "query",
			Description: "query results",
			Fields: graphql.InputObjectConfigFieldMap{
				"column": &graphql.InputObjectFieldConfig{
					Type: graphql.String,
				},
				"operator": &graphql.InputObjectFieldConfig{
					Type: graphql.String,
				},
				"value": &graphql.InputObjectFieldConfig{
					Type: graphql.String,
				},
			},
		})),
		Description:  "filter results by search query",
		DefaultValue: "",
	}

	filterArgument := graphql.ArgumentConfig{
		Type:         graphql.String,
		Description:  "filter data by keyword",
		DefaultValue: "",
	}

	for _, table := range cmsConfig.Tables {

		if len(table.TableName) < 1 {
			continue
		}
		if table.IsJoinTable {
			continue
		}

		tableType := graphql.NewObject(graphql.ObjectConfig{
			Name: table.TableName,
			Interfaces: []*graphql.Interface{
				nodeDefinitions.NodeInterface,
			},
			Fields:      graphql.Fields{},
			Description: table.TableName,
		})

		inputTypesMap[table.TableName] = tableType

	}

	tableColumnMap := map[string]map[string]api2go.ColumnInfo{}

	for _, table := range cmsConfig.Tables {
		if table.IsJoinTable {
			continue
		}
		columnMap := map[string]api2go.ColumnInfo{}

		for _, col := range table.Columns {
			columnMap[col.ColumnName] = col
		}
		tableColumnMap[table.TableName] = columnMap
	}

	for _, table := range cmsConfig.Tables {

		if len(table.TableName) < 1 {
			continue
		}
		if table.IsJoinTable {
			continue
		}
		allFields := make(graphql.FieldConfigArgument)
		uniqueFields := make(graphql.FieldConfigArgument)

		fields := make(graphql.Fields)

		for _, column := range table.Columns {

			allFields[table.TableName+"."+column.ColumnName] = &graphql.ArgumentConfig{
				Type:         resource.ColumnManager.GetGraphqlType(column.ColumnType),
				DefaultValue: column.DefaultValue,
				Description:  column.ColumnDescription,
			}

			if column.IsUnique || column.IsPrimaryKey {
				uniqueFields[table.TableName+"."+column.ColumnName] = &graphql.ArgumentConfig{
					Type:         resource.ColumnManager.GetGraphqlType(column.ColumnType),
					DefaultValue: column.DefaultValue,
					Description:  column.ColumnDescription,
				}
			}

			if column.IsForeignKey {
				continue
			}

			var graphqlType graphql.Type
			//if column.IsForeignKey {
			//	switch column.ForeignKeyData.DataSource {
			//	case "self":
			//		graphqlType = inputTypesMap[column.ForeignKeyData.Namespace]
			//	case "cloud_store":
			//		graphqlType = inputTypesMap[column.ForeignKeyData.Namespace]
			//	default:
			//		log.Errorf("Unknown data source of column [%s] in table [%v] cannot be defined in graphql schema %s", column.ColumnName, table.TableName, column.ForeignKeyData)
			//	}
			//} else {
			graphqlType = resource.ColumnManager.GetGraphqlType(column.ColumnType)
			//}

			fields[column.ColumnName] = &graphql.Field{
				Type:        graphqlType,
				Description: column.ColumnDescription,
			}
		}

		for _, relation := range table.Relations {

			targetName := relation.GetSubjectName()
			targetObject := relation.GetSubject()
			if relation.Subject == table.TableName {
				targetName = relation.GetObjectName()
				targetObject = relation.GetObject()
			}

			switch relation.Relation {
			case "belongs_to":
				fields[targetName] = &graphql.Field{
					Type:        graphql.NewNonNull(inputTypesMap[targetObject]),
					Description: fmt.Sprintf("Belongs to %v", relation.Subject),
				}
			case "has_one":
				fields[targetName] = &graphql.Field{
					Type:        inputTypesMap[targetObject],
					Description: fmt.Sprintf("Has one %v", relation.Subject),
				}

			case "has_many":
				listType, ok := inputTypesMap[targetObject]
				if !ok {
					log.Errorf("[247] target object has no proper input type: %v", targetObject)
				}
				fields[targetName] = &graphql.Field{
					Type:        graphql.NewList(listType),
					Description: fmt.Sprintf("Has many %v", relation.Subject),
				}

			case "has_many_and_belongs_to_many":
				fields[targetName] = &graphql.Field{
					Type:        graphql.NewList(inputTypesMap[targetObject]),
					Description: fmt.Sprintf("Related %v", relation.Subject),
				}

			}

		}

		fields["id"] = &graphql.Field{
			Description: "The ID of an object",
			Type:        graphql.NewNonNull(graphql.ID),
		}

		for fieldName, config := range fields {
			inputTypesMap[table.TableName].AddFieldConfig(fieldName, config)
		}

		// all table names query field

		rootFields[table.TableName] = &graphql.Field{
			Type:        graphql.NewList(inputTypesMap[table.TableName]),
			Description: "Find all " + table.TableName,
			Args: graphql.FieldConfigArgument{
				"filter": &filterArgument,
				"query":  &queryArgument,
				"page":   &pageConfig,
			},
			//Args:        uniqueFields,
			Resolve: func(table table_info.TableInfo) func(params graphql.ResolveParams) (interface{}, error) {
				return func(params graphql.ResolveParams) (interface{}, error) {

					//log.Printf("Arguments: %v", params.Args)

					filters := make([]resource.Query, 0)

					query, isQueried := params.Args["query"]
					if isQueried {
						queryMap, ok := query.([]interface{})
						if ok {
							for _, qu := range queryMap {
								q := qu.(map[string]interface{})
								query := resource.Query{
									ColumnName: q["column"].(string),
									Operator:   q["operator"].(string),
									Value:      q["value"].(string),
								}
								filters = append(filters, query)
							}
						}
					}

					filter, isFiltered := params.Args["filter"]

					if !isFiltered {
						filter = ""
					}

					ur, _ := url.Parse("/api/" + table.TableName)
					pr := &http.Request{
						Method: "GET",
						URL:    ur,
					}
					pr = pr.WithContext(params.Context)

					pageNumber := 1
					pageSize := 10
					pageParams, ok := params.Args["page"]
					if ok {
						pageParamsMap, ok := pageParams.(map[string]interface{})
						if ok {
							pageSizeNew, ok := pageParamsMap["size"]
							if ok {
								pageSize, ok = pageSizeNew.(int)
							}
							pageNumberNew, ok := pageParamsMap["number"]
							if ok {
								pageNumber, ok = pageNumberNew.(int)
							}
						}

					}

					jsStr, err := json.Marshal(filters)
					req := api2go.Request{
						PlainRequest: pr,

						QueryParams: map[string][]string{
							"query":              {string(jsStr)},
							"filter":             {filter.(string)},
							"page[number]":       {fmt.Sprintf("%v", pageNumber)},
							"page[size]":         {fmt.Sprintf("%v", pageSize)},
							"included_relations": {"*"},
						},
					}

					_, responder, err := resources[table.TableName].PaginatedFindAll(req)

					if err != nil {
						return nil, fmt.Errorf("no such entity - [%v]", table.TableName)
					}
					items := make([]map[string]interface{}, 0)
					results := responder.Result().([]api2go.Api2GoModel)

					if responder.Result() == nil {
						return results, nil

					}

					columnMap := tableColumnMap[table.TableName]

					for _, r := range results {

						included := r.Includes
						includedMap := make(map[string]jsonapi.MarshalIdentifier)

						for _, included := range included {
							includedMap[included.GetID()] = included
						}

						data := r.GetAttributes()

						for key, val := range data {
							colInfo, ok := columnMap[key]
							if !ok {
								continue
							}

							strVal, ok := val.(daptinid.DaptinReferenceId)
							if !ok {
								continue
							}

							if colInfo.IsForeignKey {
								fObj, ok := includedMap[strVal.String()]

								if ok {
									data[key] = fObj.GetAttributes()
								}
							}

						}

						items = append(items, data)

					}

					return items, err

				}
			}(table),
		}

		rootFields["aggregate"+strcase.ToCamel(table.TableName)] = &graphql.Field{
			Type:        graphql.NewList(inputTypesMap[table.TableName]),
			Description: "Aggregates for " + strings.ReplaceAll(table.TableName, "_", " "),
			Args: graphql.FieldConfigArgument{
				"group": &graphql.ArgumentConfig{
					Type: graphql.NewList(graphql.String),
				},
				"having": &graphql.ArgumentConfig{
					Type: graphql.NewList(graphql.String),
				},
				"filter": &graphql.ArgumentConfig{
					Type: graphql.NewList(graphql.String),
				},
				"join": &graphql.ArgumentConfig{
					Type: graphql.NewList(graphql.String),
				},
				"column": &graphql.ArgumentConfig{
					Type: graphql.NewList(graphql.String),
				},
				"order": &graphql.ArgumentConfig{
					Type: graphql.NewList(graphql.String),
				},
			},
			Resolve: func(table table_info.TableInfo) func(params graphql.ResolveParams) (interface{}, error) {

				return func(params graphql.ResolveParams) (interface{}, error) {
					log.Printf("GraphQL Aggregate Query Arguments: %v", params.Args)

					user := params.Context.Value("user")
					var sessionUser *auth.SessionUser
					if user != nil {
						sessionUser = user.(*auth.SessionUser)
					}

					transaction, err := resources[table.TableName].Connection().Beginx()
					if err != nil {
						resource.CheckErr(err, "Failed to begin transaction [548]")
						return nil, err
					}

					defer transaction.Commit()
					perm := resources[table.TableName].GetObjectPermissionByWhereClause("world", "table_name", table.TableName, transaction)
					if sessionUser == nil || !perm.CanExecute(sessionUser.UserReferenceId, sessionUser.Groups, resources[table.TableName].AdministratorGroupId) {
						return nil, errors.New("unauthorized")
					}

					aggReq := resource.AggregationRequest{}

					aggReq.RootEntity = table.TableName

					if params.Args["group"] != nil {
						groupBys := params.Args["group"].([]interface{})
						aggReq.GroupBy = make([]string, 0)
						for _, grp := range groupBys {
							aggReq.GroupBy = append(aggReq.GroupBy, grp.(string))
						}
					}

					if params.Args["filter"] != nil {
						filters := params.Args["filter"].([]interface{})
						aggReq.Filter = make([]string, 0)
						for _, grp := range filters {
							aggReq.Filter = append(aggReq.GroupBy, grp.(string))
						}
					}

					if params.Args["having"] != nil {
						havingClauseList := params.Args["having"].([]interface{})
						aggReq.Having = make([]string, 0)
						for _, grp := range havingClauseList {
							aggReq.Having = append(aggReq.GroupBy, grp.(string))
						}
					}

					if params.Args["join"] != nil {
						groupBys := params.Args["join"].([]interface{})
						aggReq.Join = make([]string, 0)
						for _, grp := range groupBys {
							aggReq.Join = append(aggReq.Join, grp.(string))
						}
					}
					if params.Args["column"] != nil {
						groupBys := params.Args["column"].([]interface{})
						aggReq.ProjectColumn = make([]string, 0)
						for _, grp := range groupBys {
							aggReq.ProjectColumn = append(aggReq.ProjectColumn, grp.(string))
						}
					}
					if params.Args["order"] != nil {
						groupBys := params.Args["order"].([]interface{})
						aggReq.Order = make([]string, 0)
						for _, grp := range groupBys {
							aggReq.Order = append(aggReq.Order, grp.(string))
						}
					}

					//params.Args["query"].(string)
					//aggReq.Query =

					aggResponse, err := resources[table.TableName].DataStats(aggReq, transaction)

					return aggResponse.Data, err
				}
			}(table),
		}
	}
	rootFields["node"] = nodeDefinitions.NodeField

	//rootQuery := graphql.NewObject(graphql.ObjectConfig{
	//	Name:   "RootQuery",
	//	Fields: rootFields,
	//})

	// root query
	// we just define a trivial example here, since root query is required.
	// Test with curl
	// curl -g 'http://localhost:8080/graphql?query={lastTodo{id,text,done}}'
	var rootQuery = graphql.NewObject(graphql.ObjectConfig{
		Name:   "RootQuery",
		Fields: rootFields,
	})

	for _, t := range cmsConfig.Tables {
		if t.IsJoinTable {
			continue
		}

		func(table table_info.TableInfo) {

			inputFields := make(graphql.FieldConfigArgument)
			updateFields := make(graphql.FieldConfigArgument)

			for _, col := range table.Columns {

				if resource.IsStandardColumn(col.ColumnName) {
					continue
				}
				if col.IsForeignKey {
					continue
				}

				var finalGraphqlType graphql.Type
				var finalGraphqlType1 graphql.Type
				finalGraphqlType = resource.ColumnManager.GetGraphqlType(col.ColumnType)
				finalGraphqlType1 = finalGraphqlType

				updateFields[col.ColumnName] = &graphql.ArgumentConfig{
					Type:         finalGraphqlType,
					Description:  col.ColumnDescription,
					DefaultValue: col.DefaultValue,
				}

				if !col.IsNullable || col.ColumnType == "encrypted" {
					finalGraphqlType1 = graphql.NewNonNull(finalGraphqlType)
				}

				inputFields[col.ColumnName] = &graphql.ArgumentConfig{
					Type:         finalGraphqlType1,
					Description:  col.ColumnDescription,
					DefaultValue: col.DefaultValue,
				}

			}

			mutationFields["add"+strcase.ToCamel(table.TableName)] = &graphql.Field{
				Type:        inputTypesMap[table.TableName],
				Description: "Create new " + strings.ReplaceAll(table.TableName, "_", " "),
				Args:        inputFields,
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					obj := api2go.NewApi2GoModelWithData(table.TableName, nil, 0, nil, params.Args)

					ur, _ := url.Parse("/api/" + table.TableName)

					pr := &http.Request{
						Method: "POST",
						URL:    ur,
					}

					pr = pr.WithContext(params.Context)

					req := api2go.Request{
						PlainRequest: pr,
					}
					transaction, err := resources[table.TableName].Connection().Beginx()
					if err != nil {
						return nil, err
					}
					defer transaction.Commit()

					created, err := resources[table.TableName].CreateWithTransaction(obj, req, transaction)

					if err != nil {
						return nil, err
					}

					return created.Result().(api2go.Api2GoModel).GetAttributes(), err
				},
			}

			updateInputFields := make(graphql.FieldConfigArgument)
			for k, v := range updateFields {
				updateInputFields[k] = v
			}

			updateInputFields["reference_id"] = &graphql.ArgumentConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Resource id",
			}

			mutationFields["update"+strcase.ToCamel(table.TableName)] = &graphql.Field{
				Type:        inputTypesMap[table.TableName],
				Description: "Update " + strings.ReplaceAll(table.TableName, "_", " "),
				Args:        updateInputFields,
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {

					referenceIdInf, ok := params.Args["reference_id"]
					var referenceId daptinid.DaptinReferenceId
					if ok {
						referenceId = daptinid.DaptinReferenceId(uuid.MustParse(referenceIdInf.(string)))
					} else {
						log.Errorf("parameter reference_id is not a valid string")
						return nil, errors.New("invalid parameter value for reference_id")
					}

					sessionUser := &auth.SessionUser{}
					sessionUserInterface := params.Context.Value("user")
					if sessionUserInterface != nil {
						sessionUser = sessionUserInterface.(*auth.SessionUser)
					}

					transaction, err := resources[table.TableName].Connection().Beginx()
					if err != nil {
						return nil, err
					}
					defer func() {
						err = transaction.Rollback()
						if err != nil {
							log.Debugf("Failed to rollback: %v", err)
						}
					}()

					existingObj, _, err := resources[table.TableName].GetSingleRowByReferenceIdWithTransaction(table.TableName,
						referenceId, nil, transaction)
					log.Tracef("Completed mutationFields GetSingleRowByReferenceIdWithTransaction")
					if err != nil {
						return nil, err
					}

					log.Printf("Get row permission before update: %v", existingObj)
					permission := resources[table.TableName].GetRowPermissionWithTransaction(existingObj, transaction)

					if !permission.CanPeek(sessionUser.UserReferenceId, sessionUser.Groups, resources[table.TableName].AdministratorGroupId) {
						return nil, errors.New("unauthorized")
					}

					obj := api2go.NewApi2GoModelWithData(table.TableName, nil, 0, nil, existingObj)

					args := params.Args
					deleteKeys := make([]string, 0)
					for k := range args {
						if args[k] == "" {
							deleteKeys = append(deleteKeys, k)
						}
					}

					for _, s := range deleteKeys {
						delete(args, s)
					}

					delete(args, "reference_id")

					obj.SetAttributes(args)
					ur, _ := url.Parse("/api/" + table.TableName)

					pr := &http.Request{
						Method: "PATCH",
						URL:    ur,
					}

					pr = pr.WithContext(params.Context)

					req := api2go.Request{
						PlainRequest: pr,
					}

					created, err := resources[table.TableName].UpdateWithTransaction(obj, req, transaction)

					if err != nil {
						log.Printf("Failed to update resource: %v", err)
						return nil, err
					}
					err = transaction.Commit()

					return created.Result().(api2go.Api2GoModel).GetAttributes(), err
				},
			}

			mutationFields["delete"+strcase.ToCamel(table.TableName)] = &graphql.Field{
				Type:        inputTypesMap[table.TableName],
				Description: "Delete " + strings.ReplaceAll(table.TableName, "_", " "),
				Args: graphql.FieldConfigArgument{
					"reference_id": &graphql.ArgumentConfig{
						Type:        graphql.String,
						Description: "Resource id",
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {

					ur, _ := url.Parse("/api/" + table.TableName)
					pr := &http.Request{
						Method: "DELETE",
						URL:    ur,
					}

					pr = pr.WithContext(params.Context)

					req := api2go.Request{
						PlainRequest: pr,
					}

					transaction, err := resources[table.TableName].Connection().Beginx()
					if err != nil {
						return nil, err
					}
					defer transaction.Commit()

					_, err = resources[table.TableName].DeleteWithTransaction(daptinid.DaptinReferenceId(uuid.MustParse(params.Args["reference_id"].(string))), req, transaction)

					if err != nil {
						return nil, err
					}

					return fmt.Sprintf(`{
													"data": {
														"delete%s": {
														}
													}
												}`, flect.Capitalize(table.TableName)), err
				},
			}

		}(t)

	}

	for _, a := range cmsConfig.Actions {

		func(action actionresponse.Action) {

			inputFields := make(graphql.FieldConfigArgument)

			for _, col := range action.InFields {

				var finalGraphqlType graphql.Type
				finalGraphqlType = resource.ColumnManager.GetGraphqlType(col.ColumnType)

				if !col.IsNullable {
					finalGraphqlType = graphql.NewNonNull(finalGraphqlType)
				}

				inputFields[col.ColumnName] = &graphql.ArgumentConfig{
					Type:         finalGraphqlType,
					Description:  col.ColumnDescription,
					DefaultValue: col.DefaultValue,
				}

			}

			//if !action.InstanceOptional {
			//	inputFields[action.OnType+"_id"] = &graphql.ArgumentConfig{
			//		Type:        graphql.NewNonNull(graphql.String),
			//		Description: "reference id of subject " + action.OnType,
			//	}
			//}

			mutationFields["execute"+strcase.ToCamel(action.Name)+"On"+strcase.ToCamel(action.OnType)] = &graphql.Field{
				Type:        graphql.NewList(actionResponseType),
				Description: "Execute " + strings.ReplaceAll(action.Name, "_", " ") + " on " + action.OnType,
				Args:        inputFields,
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {

					ur, _ := url.Parse("/action/" + action.OnType + "/" + action.Name)
					pr := &http.Request{
						Method: "EXECUTE",
						URL:    ur,
					}

					pr = pr.WithContext(params.Context)

					req := api2go.Request{
						PlainRequest: pr,
					}

					actionRequest := actionresponse.ActionRequest{
						Type:       action.OnType,
						Action:     action.Name,
						Attributes: params.Args,
					}

					transaction, err := resources[action.OnType].Connection().Beginx()
					if err != nil {
						return nil, err
					}
					defer transaction.Commit()

					response, err := resources[action.OnType].HandleActionRequest(actionRequest, req, transaction)
					if err != nil {
						transaction.Rollback()
					}

					return response, err
				},
			}
		}(a)

	}

	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name:   "Mutation",
		Fields: mutationFields,
	})

	var err error
	Schema, err = graphql.NewSchema(graphql.SchemaConfig{
		Query:    rootQuery,
		Mutation: mutationType,
	})
	if err != nil {
		log.Errorf("Failed to generate graphql schema: %v", err)
	}

	return &Schema

	//for _, table := range cmsConfig.Tables {
	//
	//	for _, relation := range table.Relations {
	//		if relation.Relation == "has_one" || relation.Relation == "belongs_to" {
	//			if relation.Subject == table.TableName {
	//				log.Printf("Add column: %v", table.TableName+"."+relation.GetObjectName())
	//				if done[table.TableName+"."+relation.GetObjectName()] {
	//					continue
	//					panic("ok")
	//				}
	//				done[table.TableName+"."+relation.GetObjectName()] = true
	//				inputTypesMap[table.TableName].Fields()[table.TableName+"."+relation.GetObjectName()] = &graphql.InputObjectField{
	//					Type:        inputTypesMap[relation.GetObject()],
	//					PrivateName: relation.GetObjectName(),
	//				}
	//			} else {
	//				log.Printf("Add column: %v", table.TableName+"."+relation.GetSubjectName())
	//				if done[table.TableName+"."+relation.GetSubjectName()] {
	//					// panic("ok")
	//				}
	//				done[table.TableName+"."+relation.GetSubjectName()] = true
	//				inputTypesMap[table.TableName].Fields()[table.TableName+"."+relation.GetSubjectName()] = &graphql.InputObjectField{
	//					Type:        inputTypesMap[relation.GetSubject()],
	//					PrivateName: relation.GetSubjectName(),
	//				}
	//			}
	//
	//		} else {
	//			if relation.Subject == table.TableName {
	//				log.Printf("Add column: %v", table.TableName+"."+relation.GetObjectName())
	//				if done[table.TableName+"."+relation.GetObjectName()] {
	//					panic("ok")
	//				}
	//				done[table.TableName+"."+relation.GetObjectName()] = true
	//				inputTypesMap[table.TableName].Fields()[table.TableName+"."+relation.GetObjectName()] = &graphql.InputObjectField{
	//					PrivateName: relation.GetObjectName(),
	//					Type:        graphql.NewList(inputTypesMap[relation.GetObject()]),
	//				}
	//			} else {
	//				log.Printf("Add column: %v", table.TableName+"."+relation.GetSubjectName())
	//				if done[table.TableName+"."+relation.GetSubjectName()] {
	//					panic("ok")
	//				}
	//				done[table.TableName+"."+relation.GetSubjectName()] = true
	//				inputTypesMap[table.TableName].Fields()[table.TableName+"."+relation.GetSubjectName()] = &graphql.InputObjectField{
	//					Type:        graphql.NewList(inputTypesMap[relation.GetSubject()]),
	//					PrivateName: relation.GetSubjectName(),
	//				}
	//			}
	//		}
	//	}
	//}

	//for _, table := range cmsConfig.Tables {
	//
	//	createFields := make(graphql.FieldConfigArgument)
	//
	//	for _, column := range table.Columns {
	//
	//		if column.IsForeignKey {
	//			continue
	//		}
	//
	//

	//
	//		if IsStandardColumn(column.ColumnName) {
	//			continue
	//		}
	//
	//		if column.IsForeignKey {
	//			continue
	//		}
	//
	//		if column.IsNullable {
	//			createFields[table.TableName+"."+column.ColumnName] = &graphql.ArgumentConfig{
	//				Type: resource.ColumnManager.GetGraphqlType(column.ColumnType),
	//			}
	//		} else {
	//			createFields[table.TableName+"."+column.ColumnName] = &graphql.ArgumentConfig{
	//				Type: graphql.NewNonNull(resource.ColumnManager.GetGraphqlType(column.ColumnType)),
	//			}
	//		}
	//
	//	}
	//
	//	//for _, relation := range table.Relations {
	//	//
	//	//	if relation.Relation == "has_one" || relation.Relation == "belongs_to" {
	//	//		if relation.Subject == table.TableName {
	//	//			allFields[table.TableName+"."+relation.GetObjectName()] = &graphql.ArgumentConfig{
	//	//				Type: inputTypesMap[relation.GetObject()],
	//	//			}
	//	//		} else {
	//	//			allFields[table.TableName+"."+relation.GetSubjectName()] = &graphql.ArgumentConfig{
	//	//				Type: inputTypesMap[relation.GetSubject()],
	//	//			}
	//	//		}
	//	//
	//	//	} else {
	//	//		if relation.Subject == table.TableName {
	//	//			allFields[table.TableName+"."+relation.GetObjectName()] = &graphql.ArgumentConfig{
	//	//				Type: graphql.NewList(inputTypesMap[relation.GetObject()]),
	//	//			}
	//	//		} else {
	//	//			allFields[table.TableName+"."+relation.GetSubjectName()] = &graphql.ArgumentConfig{
	//	//				Type: graphql.NewList(inputTypesMap[relation.GetSubject()]),
	//	//			}
	//	//		}
	//	//	}
	//	//}
	//
	//	//mutations["create"+Capitalize(table.TableName)] = &graphql.InputObjectFieldConfig{
	//	//	Type:        inputTypesMap[table.TableName],
	//	//	Description: "Create a new " + table.TableName,
	//	//	//Args:        createFields,
	//	//	//Resolve: func(table resource.TableInfo) (func(params graphql.ResolveParams) (interface{}, error)) {
	//	//	//	return func(p graphql.ResolveParams) (interface{}, error) {
	//	//	//		log.Printf("create resolve params: %v", p)
	//	//	//
	//	//	//		data := make(map[string]interface{})
	//	//	//
	//	//	//		for key, val := range p.Args {
	//	//	//			data[key] = val
	//	//	//		}
	//	//	//
	//	//	//		model := api2go.NewApi2GoModelWithData(table.TableName, nil, 0, nil, data)
	//	//	//
	//	//	//		pr := &http.Request{
	//	//	//			Method: "PATCH",
	//	//	//		}
	//	//	//		pr = pr.WithContext(p.Context)
	//	//	//		req := api2go.Request{
	//	//	//			PlainRequest: pr,
	//	//	//			QueryParams: map[string][]string{
	//	//	//			},
	//	//	//		}
	//	//	//
	//	//	//		res, err := resources[table.TableName].Create(model, req)
	//	//	//
	//	//	//		return res.Result().(api2go.Api2GoModel).Data, err
	//	//	//	}
	//	//	//}(table),
	//	//}
	//
	//	//mutations["update"+Capitalize(table.TableName)] = &graphql.InputObjectFieldConfig{
	//	//	Type:        inputTypesMap[table.TableName],
	//	//	Description: "Create a new " + table.TableName,
	//	//	//Args:        createFields,
	//	//	//Resolve: func(table resource.TableInfo) (func(params graphql.ResolveParams) (interface{}, error)) {
	//	//	//	return func(p graphql.ResolveParams) (interface{}, error) {
	//	//	//		log.Printf("create resolve params: %v", p)
	//	//	//
	//	//	//		data := make(map[string]interface{})
	//	//	//
	//	//	//		for key, val := range p.Args {
	//	//	//			data[key] = val
	//	//	//		}
	//	//	//
	//	//	//		model := api2go.NewApi2GoModelWithData(table.TableName, nil, 0, nil, data)
	//	//	//
	//	//	//		pr := &http.Request{
	//	//	//			Method: "PATCH",
	//	//	//		}
	//	//	//		pr = pr.WithContext(p.Context)
	//	//	//		req := api2go.Request{
	//	//	//			PlainRequest: pr,
	//	//	//			QueryParams: map[string][]string{
	//	//	//			},
	//	//	//		}
	//	//	//
	//	//	//		res, err := resources[table.TableName].Update(model, req)
	//	//	//
	//	//	//		return res.Result().(api2go.Api2GoModel).Data, err
	//	//	//	}
	//	//	//}(table),
	//	//}
	//

	//
	//var rootMutation = graphql.NewObject(graphql.ObjectConfig{
	//	Name:   "RootMutation",
	//	Fields: mutations,
	//});
	//var rootQuery = graphql.NewObject(graphql.ObjectConfig{
	//	Name:   "RootQuery",
	//	Fields: query,
	//})
	//
	//// define schema, with our rootQuery and rootMutation
	//var schema, _ = graphql.NewSchema(graphql.SchemaConfig{
	//	Query:    rootQuery,
	//	Mutation: rootMutation,
	//})
	//
	//return &schema

}
