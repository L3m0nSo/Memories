package resource

import (
	"fmt"
	"strings"

	"github.com/artpar/api2go/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

	//log "github.com/sirupsen/logrus"
	//"github.com/Masterminds/squirrel"
	"errors"

	"github.com/L3m0nSo/Memories/server/auth"
)

// The TableAccessPermissionChecker middleware is resposible for entity level authorization check, before and after the changes
type TableAccessPermissionChecker struct {
}

func (pc *TableAccessPermissionChecker) String() string {
	return "TableAccessPermissionChecker"
}

var errorMsgFormat = "[%v] [%v] access not allowed for action [%v] to user [%v]"

// Intercept after check implements if the data should be returned after the data change is complete
func (pc *TableAccessPermissionChecker) InterceptAfter(dr *DbResource, req *api2go.Request, results []map[string]interface{}, transaction *sqlx.Tx) ([]map[string]interface{}, error) {

	if results == nil || len(results) < 1 {
		return results, nil
	}

	//returnMap := make([]map[string]interface{}, 0)

	user := req.PlainRequest.Context().Value("user")
	sessionUser := &auth.SessionUser{}

	if user != nil {
		sessionUser = user.(*auth.SessionUser)
	}

	if IsAdminWithTransaction(sessionUser, transaction) {
		return results, nil
	}

	tableOwnership := dr.GetObjectPermissionByWhereClauseWithTransaction("world", "table_name", dr.model.GetName(), transaction)

	//log.Printf("Row Permission for [%v] for [%v]", dr.model.GetName(), tableOwnership)
	if req.PlainRequest.Method == "GET" {
		if tableOwnership.CanPeek(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) {
			//returnMap = append(returnMap, result)
			//includedMapCache[referenceId] = true
			return results, nil
		} else {
			// not allowed
		}
	} else if tableOwnership.CanPeek(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) {
		//log.Printf("[TableAccessPermissionChecker] Result not to be included: %v", result["reference_id"])
		//returnMap = append(returnMap, result)
		//includedMapCache[referenceId] = true
		return results, nil
	}

	log.Tracef("TableAccessPermissionChecker.InterceptAfter[%v] Disallowed: [%v]", tableOwnership, sessionUser)
	return nil, api2go.NewHTTPError(errors.New(fmt.Sprintf(errorMsgFormat, dr.tableInfo.TableName, req.PlainRequest.Method, sessionUser.UserReferenceId)), pc.String(), 403)
}

var (
	// Error Unauthorized
	ErrUnauthorized = errors.New("forbidden")
)

// Intercept before implemetation for entity level authentication check
func (pc *TableAccessPermissionChecker) InterceptBefore(dr *DbResource, req *api2go.Request,
	results []map[string]interface{}, transaction *sqlx.Tx) ([]map[string]interface{}, error) {

	//var err error
	//log.Printf("context: %v", context.GetAll(req.PlainRequest))

	user := req.PlainRequest.Context().Value("user")
	sessionUser := &auth.SessionUser{}

	if user != nil {
		sessionUser = user.(*auth.SessionUser)
	}

	if IsAdminWithTransaction(sessionUser, transaction) {
		return results, nil
	}

	//log.Printf("User Id: %v", sessionUser.UserReferenceId)
	//log.Printf("User Groups: %v", sessionUser.Groups)

	tableOwnership := dr.GetObjectPermissionByWhereClauseWithTransaction("world", "table_name", dr.model.GetName(), transaction)

	//log.Printf("Table owner: %v", tableOwnership.UserId)
	//log.Printf("Table groups: %v", tableOwnership.UserGroupId)

	//log.Printf("[TableAccessPermissionChecker] PermissionInstance check for type: [%v] on [%v] @%v", req.PlainRequest.Method, dr.model.GetName(), tableOwnership)
	if req.PlainRequest.Method == "GET" {
		if !tableOwnership.CanPeek(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) {
			log.Tracef("TableAccessPermissionChecker.InterceptBefore[%v] Disallowed: [%v]", tableOwnership, sessionUser)
			return nil, api2go.NewHTTPError(fmt.Errorf(errorMsgFormat, "table", dr.tableInfo.TableName, req.PlainRequest.Method, sessionUser.UserReferenceId), pc.String(), 403)
		}
	} else if req.PlainRequest.Method == "PUT" || req.PlainRequest.Method == "PATCH" {
		if strings.Index(req.PlainRequest.URL.String(), "/relationships/") > -1 {
			if !tableOwnership.CanRefer(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) ||
				!tableOwnership.CanPeek(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) {
				log.Tracef("TableAccessPermissionChecker.InterceptBefore[%v] Disallowed: [%v]", tableOwnership, sessionUser)
				return nil, api2go.NewHTTPError(fmt.Errorf(errorMsgFormat, "table", dr.tableInfo.TableName, req.PlainRequest.Method, sessionUser.UserReferenceId), pc.String(), 403)
			}
		} else {
			if !tableOwnership.CanUpdate(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) {
				log.Tracef("TableAccessPermissionChecker.InterceptBefore[%v] Disallowed: [%v]", tableOwnership, sessionUser)
				return nil, api2go.NewHTTPError(fmt.Errorf(errorMsgFormat, "table", dr.tableInfo.TableName, req.PlainRequest.Method, sessionUser.UserReferenceId), pc.String(), 403)
			}
		}

	} else if req.PlainRequest.Method == "POST" {
		if strings.Index(req.PlainRequest.URL.String(), "/relationships/") > -1 {
			if !tableOwnership.CanRefer(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) ||
				!tableOwnership.CanPeek(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) {
				log.Tracef("TableAccessPermissionChecker.InterceptBefore[%v] Disallowed: [%v]", tableOwnership, sessionUser)
				return nil, api2go.NewHTTPError(fmt.Errorf(errorMsgFormat, "table", dr.tableInfo.TableName, req.PlainRequest.Method, sessionUser.UserReferenceId), pc.String(), 403)
			}
		} else {
			if !tableOwnership.CanCreate(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) {
				log.Tracef("TableAccessPermissionChecker.InterceptBefore[%v] Disallowed: [%v]", tableOwnership, sessionUser)
				return nil, api2go.NewHTTPError(fmt.Errorf(errorMsgFormat, "table", dr.tableInfo.TableName, req.PlainRequest.Method, sessionUser.UserReferenceId), pc.String(), 403)
			}
		}
	} else if req.PlainRequest.Method == "DELETE" {
		if strings.Index(req.PlainRequest.URL.String(), "/relationships/") > -1 {
			if !tableOwnership.CanRefer(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) ||
				!tableOwnership.CanPeek(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) {
				log.Tracef("TableAccessPermissionChecker.InterceptBefore[%v] Disallowed: [%v]", tableOwnership, sessionUser)
				return nil, api2go.NewHTTPError(fmt.Errorf(errorMsgFormat, "table", dr.tableInfo.TableName, req.PlainRequest.Method, sessionUser.UserReferenceId), pc.String(), 403)
			}
		} else {
			if !tableOwnership.CanDelete(sessionUser.UserReferenceId, sessionUser.Groups, dr.AdministratorGroupId) {
				log.Tracef("TableAccessPermissionChecker.InterceptBefore[%v] Disallowed: [%v]", tableOwnership, sessionUser)
				return nil, api2go.NewHTTPError(fmt.Errorf(errorMsgFormat, "table", dr.tableInfo.TableName, req.PlainRequest.Method, sessionUser.UserReferenceId), pc.String(), 403)
			}
		}
	} else {
		log.Tracef("TableAccessPermissionChecker.InterceptBefore[%v] Disallowed: [%v]", tableOwnership, sessionUser)
		return nil, api2go.NewHTTPError(fmt.Errorf(errorMsgFormat, "table", dr.tableInfo.TableName, req.PlainRequest.Method, sessionUser.UserReferenceId), pc.String(), 403)
	}

	return results, nil

}
