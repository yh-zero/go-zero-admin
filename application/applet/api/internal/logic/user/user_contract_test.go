package user

import (
	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/applet/api/internal/types"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestUpdatePreservesProvidedEmptyValues(t *testing.T) {
	request := httptest.NewRequest("PUT", "/v1/sys/updateUserInfo", strings.NewReader(`{"ID":1,"phone":"","email":"","authorityIds":[]}`))
	request.Header.Set("Content-Type", "application/json")
	var input types.UpdateUserInfoRequest
	if err := httpx.Parse(request, &input); err != nil {
		t.Fatal(err)
	}
	result := userUpdateRequest(&input)
	if result.UserInfo.Phone != "" || result.UserInfo.Email != "" || !result.UpdateAuthorities {
		t.Fatalf("lost explicit empty fields: %+v", result)
	}
	if !reflect.DeepEqual(result.UpdateFields, []string{"phone", "email"}) {
		t.Fatalf("unexpected update fields: %v", result.UpdateFields)
	}
	omitted := userUpdateRequest(&types.UpdateUserInfoRequest{ID: 1})
	if omitted.UpdateAuthorities || len(omitted.UpdateFields) != 0 {
		t.Fatal("omitted fields must not update user")
	}
}
func TestQueryContracts(t *testing.T) {
	cases := []struct {
		url    string
		target any
		check  func() bool
	}{}
	menu := types.GetMenuAuthorityRequest{}
	cases = append(cases, struct {
		url    string
		target any
		check  func() bool
	}{"/?authorityId=42", &menu, func() bool { return menu.AuthorityId == 42 }})
	detail := types.GetBaseMenuByIdRequest{}
	cases = append(cases, struct {
		url    string
		target any
		check  func() bool
	}{"/?id=7", &detail, func() bool { return detail.Id == 7 }})
	item := types.GetSysDictionaryInfoListDetailsByIdRequest{}
	cases = append(cases, struct {
		url    string
		target any
		check  func() bool
	}{"/?id=8", &item, func() bool { return item.ID == 8 }})
	list := types.GetSysDictionaryInfoListRequest{}
	cases = append(cases, struct {
		url    string
		target any
		check  func() bool
	}{"/?sysDictionaryID=9&value=0", &list, func() bool { return list.SysDictionaryID == 9 && list.Value != nil && *list.Value == 0 }})
	for _, c := range cases {
		if err := httpx.Parse(httptest.NewRequest("GET", c.url, nil), c.target); err != nil {
			t.Fatal(err)
		}
		if !c.check() {
			t.Fatalf("query not bound: %s", c.url)
		}
	}
}
