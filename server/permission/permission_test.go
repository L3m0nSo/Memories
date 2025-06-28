package permission

import (
	"fmt"
	"testing"

	"github.com/L3m0nSo/Memories/server/auth"
	daptinid "github.com/L3m0nSo/Memories/server/id"
	"github.com/google/uuid"
)

func TestPermissionValues(t *testing.T) {
	fmt.Printf("Permissoin [%v] == %d\n", "None", auth.None)
	fmt.Printf("Permissoin [%v] == %d\n", "GuestPeek", auth.GuestPeek)
	fmt.Printf("Permissoin [%v] == %d\n", "GuestRead", auth.GuestRead)
	fmt.Printf("Permissoin [%v] == %d\n", "GuestCreate", auth.GuestCreate)
	fmt.Printf("Permissoin [%v] == %d\n", "GuestUpdate", auth.GuestUpdate)
	fmt.Printf("Permissoin [%v] == %d\n", "GuestDelete", auth.GuestDelete)
	fmt.Printf("Permissoin [%v] == %d\n", "GuestExecute", auth.GuestExecute)
	fmt.Printf("Permissoin [%v] == %d\n", "GuestRefer", auth.GuestRefer)
	fmt.Printf("Permissoin [%v] == %d\n", "UserPeek", auth.UserPeek)
	fmt.Printf("Permissoin [%v] == %d\n", "UserRead", auth.UserRead)
	fmt.Printf("Permissoin [%v] == %d\n", "UserCreate", auth.UserCreate)
	fmt.Printf("Permissoin [%v] == %d\n", "UserUpdate", auth.UserUpdate)
	fmt.Printf("Permissoin [%v] == %d\n", "UserDelete", auth.UserDelete)
	fmt.Printf("Permissoin [%v] == %d\n", "UserExecute", auth.UserExecute)
	fmt.Printf("Permissoin [%v] == %d\n", "UserRefer", auth.UserRefer)
	fmt.Printf("Permissoin [%v] == %d\n", "GroupPeek", auth.GroupPeek)
	fmt.Printf("Permissoin [%v] == %d\n", "GroupRead", auth.GroupRead)
	fmt.Printf("Permissoin [%v] == %d\n", "GroupCreate", auth.GroupCreate)
	fmt.Printf("Permissoin [%v] == %d\n", "GroupUpdate", auth.GroupUpdate)
	fmt.Printf("Permissoin [%v] == %d\n", "GroupDelete", auth.GroupDelete)
	fmt.Printf("Permissoin [%v] == %d\n", "GroupExecute", auth.GroupExecute)
	fmt.Printf("Permissoin [%v] == %d\n", "GroupRefer", auth.GroupRefer)

}

func TestPermission(t *testing.T) {

	pi := PermissionInstance{
		UserId: daptinid.DaptinReferenceId(uuid.New()),
		UserGroupId: auth.GroupPermissionList{
			{
				GroupReferenceId:    daptinid.DaptinReferenceId(uuid.New()),
				ObjectReferenceId:   daptinid.NullReferenceId,
				RelationReferenceId: daptinid.NullReferenceId,
				Permission:          auth.UserRead | auth.GroupCRUD | auth.GroupExecute,
			},
		},
		Permission: auth.GroupCreate,
	}

	pi.CanCreate(daptinid.DaptinReferenceId(uuid.New()), auth.GroupPermissionList{
		{
			GroupReferenceId:    daptinid.DaptinReferenceId(uuid.New()),
			ObjectReferenceId:   daptinid.NullReferenceId,
			RelationReferenceId: daptinid.NullReferenceId,
			Permission:          auth.GuestRead | auth.GroupCRUD | auth.GroupExecute,
		},
	}, daptinid.NullReferenceId)

}
