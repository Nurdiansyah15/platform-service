package permission

import (
	"errors"

	"platform-service/internal/app/dto"
	"platform-service/internal/domain/permission/entity"
	"platform-service/internal/domain/permission/service"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
	solutionconstant "platform-service/internal/domain/solution/constant"
)

func mapPermissionDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodePermissionNotFound, constant.CodeGroupNotFound, constant.CodeScopeTypeNotFound,
		solutionconstant.CodeSolutionNotFound:
		return apperror.NotFound(string(de.Code))
	case constant.CodeCatalogSyncForbidden:
		return apperror.Forbidden(string(de.Code))
	case constant.CodePermissionCodeTaken, constant.CodePermissionSourceConflict, constant.CodePermissionSyncManaged,
		constant.CodeGroupCodeTaken, constant.CodeGroupSourceConflict, constant.CodeGroupSyncManaged,
		constant.CodeScopeTypeCodeTaken, constant.CodeScopeTypeSourceConflict, constant.CodeScopeTypeSyncManaged:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodePermissionCodeInvalid, constant.CodePermissionNameInvalid, constant.CodePermissionCodeDuplicate,
		constant.CodeCatalogVersionInvalid,
		constant.CodeGroupCodeInvalid, constant.CodeGroupNameInvalid, constant.CodeGroupDuplicate, constant.CodeGroupUnknown,
		constant.CodeGroupParentUnknown, constant.CodeGroupParentInvalid, constant.CodeGroupTooDeep,
		constant.CodeScopeTypeCodeInvalid, constant.CodeScopeTypeReserved, constant.CodeScopeTypeNameInvalid,
		constant.CodeScopeTypeParentInvalid, constant.CodeScopeTypeDuplicate,
		constant.CodePermissionStatusInvalid:
		return apperror.Unprocessable(string(de.Code), nil, err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func isPlatformAdmin(p *principal.Principal) bool {
	return p.IsUser() && p.IsPlatformAdmin
}

// canSync — platform admin, atau token service milik solution tersebut
// (scope catalog:sync dicek di route).
func canSync(p *principal.Principal, solutionID string) bool {
	return isPlatformAdmin(p) || (p.IsService() && p.SolutionID == solutionID)
}

func isNotFound(err error) bool {
	var de *domainerr.DomainError
	return errors.As(err, &de) && de.Code == solutionconstant.CodeSolutionNotFound
}

// toPermissionResponse memetakan permission (domain) ke DTO, dengan ringkasan group-nya.
func toPermissionResponse(p *entity.Permission, groups []*entity.Group) *dto.PermissionResponse {
	resp := &dto.PermissionResponse{
		ID: p.ID, SolutionID: p.SolutionID, Code: p.Code.Value(), Name: p.Name, Description: p.Description,
		Status: p.Status.String(), Source: p.Source.String(), CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	if p.GroupID != nil {
		resp.Group = groupRef(groups, *p.GroupID)
	}
	return resp
}

// groupRef — ringkasan group beserta leluhurnya (nil bila group tidak ditemukan).
func groupRef(groups []*entity.Group, id string) *dto.PermissionGroupRef {
	path := service.GroupPath(groups, id)
	if len(path) == 0 {
		return nil
	}
	g := path[len(path)-1]
	ref := &dto.PermissionGroupRef{Code: g.Code.Value(), Name: g.Name, SortOrder: g.SortOrder,
		Parents: make([]dto.PermissionGroupAncestor, 0, len(path)-1)}
	for _, a := range path[:len(path)-1] {
		ref.Parents = append(ref.Parents, dto.PermissionGroupAncestor{Code: a.Code.Value(), Name: a.Name, SortOrder: a.SortOrder})
	}
	return ref
}

// currentGroupCode — code group permission saat ini (nil bila tanpa grup).
func currentGroupCode(p *entity.Permission, groups []*entity.Group) *string {
	if p.GroupID == nil {
		return nil
	}
	for _, g := range groups {
		if g.ID == *p.GroupID {
			c := g.Code.Value()
			return &c
		}
	}
	return nil
}

// toGroupResponse memetakan group ke DTO; groups (group solution) dipakai untuk code induk.
func toGroupResponse(g *entity.Group, groups []*entity.Group) *dto.PermissionGroupResponse {
	resp := &dto.PermissionGroupResponse{
		ID: g.ID, ParentID: g.ParentID, Code: g.Code.Value(), Name: g.Name, Description: g.Description,
		SortOrder: g.SortOrder, Status: g.Status.String(), Source: g.Source.String(),
	}
	if g.ParentID != nil {
		for _, p := range groups {
			if p.ID == *g.ParentID {
				c := p.Code.Value()
				resp.Parent = &c
				break
			}
		}
	}
	return resp
}

// sourceBuiltin — source scope type company & branch (dikelola platform).
const sourceBuiltin = "builtin"

func toScopeTypeResponse(st *entity.ScopeType) *dto.ScopeTypeResponse {
	return &dto.ScopeTypeResponse{
		ID: st.ID, Code: st.Code.Value(), Name: st.Name, Description: st.Description, Parent: st.Parent.String(),
		SortOrder: st.SortOrder, Status: st.Status.String(), Source: st.Source.String(),
	}
}
