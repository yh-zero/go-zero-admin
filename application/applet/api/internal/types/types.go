// Code generated by goctl. DO NOT EDIT.
package types

type Model struct {
	ID        int64  `json:"ID,optional"`
	CreatedAt string `json:"CreatedAt,optional"`
	UpdatedAt string `json:"UpdatedAt,optional"`
	DeletedAt string `json:"DeletedAt,optional"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type UserInfo struct {
	UUID        string         `json:"uuid"`        // 用户UUID
	Username    string         `json:"userName"`    // 用户登录名
	Password    string         `json:"-"`           // 用户登录密码
	NickName    string         `json:"nickName"`    // 用户昵称
	SideMode    string         `json:"sideMode"`    // 用户侧边主题
	HeaderImg   string         `json:"headerImg"`   // 用户头像
	BaseColor   string         `json:"baseColor"`   // 基础颜色
	ActiveColor string         `json:"activeColor"` // 活跃颜色
	AuthorityId int64          `json:"authorityId"` // 用户角色ID
	Authority   SysAuthority   `json:"authority"`
	Authorities []SysAuthority `json:"authorities"`
	Phone       string         `json:"phone"`  // 用户手机号
	Email       string         `json:"email"`  // 用户邮箱
	Enable      int64          `json:"enable"` //用户是否被冻结 1正常 2冻结
	Model
}

type SysAuthority struct {
	AuthorityId     int64           `json:"authorityId"`   // 角色ID
	AuthorityName   string          `json:"authorityName"` // 角色名
	ParentId        *int64          `json:"parentId"`      // 父角色ID
	DataAuthorityId []*SysAuthority `json:"dataAuthorityId,optional"`
	Children        []SysAuthority  `json:"children,optional"`
	SysBaseMenus    []SysBaseMenu   `json:"menus,optional"`
	DefaultRouter   string          `json:"defaultRouter,optional"` // 默认菜单(默认dashboard)
	ShowMenuIds     string          `json:"showMenuIds,optional"`
	CreatedAt       string          `json:"CreatedAt,optional"` // 创建时间
	UpdatedAt       string          `json:"UpdatedAt,optional"` // 更新时间
	DeletedAt       string          `json:"DeletedAt,optional"`
}

type SysBaseMenu struct {
	MenuLevel     int64                  `json:"-"`
	ParentId      string                 `json:"parentId"`      // 父菜单ID
	Path          string                 `json:"path"`          // 路由path
	Name          string                 `json:"name"`          // 路由name
	Hidden        bool                   `json:"hidden"`        // 是否在列表隐藏
	Component     string                 `json:"component"`     // 对应前端文件路径
	Sort          int64                  `json:"sort,optional"` // 排序标记
	SysAuthoritys []SysAuthority         `json:"authoritys,optional"`
	Children      []SysBaseMenu          `json:"children,optional"`
	Parameters    []SysBaseMenuParameter `json:"parameters"`
	MenuBtn       []SysBaseMenuBtn       `json:"menuBtn"`
	Meta          Meta                   `json:"meta"`
	Model
}

type SysBaseMenuParameter struct {
	SysBaseMenuID int64  `json:"sysBaseMenuId,optional"`
	Type          string `json:"type"`  // 地址栏携带参数为params还是query
	Key           string `json:"key"`   // 地址栏携带参数的key
	Value         string `json:"value"` // 地址栏携带参数的值
	Model
}

type SysBaseMenuBtn struct {
	Name          string `json:"name"`
	Desc          string `json:"desc"`
	SysBaseMenuID int64  `json:"sysBaseMenuID,optional"`
	Model
}

type LoginRequest struct {
	UserName string `json:"username"`
	Password string `json:"password"`
	Captcha  string `json:"captcha"`
}

type LoginResponse struct {
	AccessExpire int64    `json:"accessExpire"`
	AccessToken  string   `json:"accessToken"`
	UserInfo     UserInfo `json:"userInfo"`
}

type RandomImageRequest struct {
}

type RandomImageResponse struct {
	CaptchaImg string `json:"captchaImg"`
}

type GetMenuRequest struct {
}

type GetMenuResponse struct {
	Menus []SysMenu `json:"menus"`
}

type SysMenu struct {
	SysBaseMenu
	MenuId      string                 `json:"menuId"`
	AuthorityId int64                  `json:"-" gorm:"comment:角色ID"`
	Children    []SysMenu              `json:"children"`
	Parameters  []SysBaseMenuParameter `json:"parameters"`
	Btns        map[string]int64       `json:"btns"`
}

type Meta struct {
	ActiveName  string `json:"activeName,optional"`
	KeepAlive   bool   `json:"keepAlive,optional"`   // 是否缓存
	DefaultMenu bool   `json:"defaultMenu,optional"` // 是否是基础路由（开发中）
	Title       string `json:"title"`                // 菜单名
	Icon        string `json:"icon"`                 // 菜单图标
	CloseTab    bool   `json:"closeTab"`             // 自动关闭tab
}

type PageRequest struct {
	PageNo   int64  `form:"pageNo,optional"`
	PageSize int64  `form:"pageSize,optional"`
	Keyword  string `form:"keyword,optional"`
}

type PageResponse struct {
	Total    int64 `json:"total"`
	PageNo   int64 `json:"page"`
	PageSize int64 `json:"pageSize"`
}

type GetMenuListRequest struct {
	PageRequest
}

type GetMenuListResponse struct {
	List []SysBaseMenu `json:"list"`
	PageResponse
}

type AddBaseMenuRequest struct {
	SysBaseMenu
}

type AddBaseMenuResponse struct {
}

type GetAuthorityListRequest struct {
	PageRequest
}

type GetAuthorityListResponse struct {
	List []SysAuthority `json:"list"`
	PageResponse
}

type SysApi struct {
	Path        string `json:"path,optional" form:"path,optional"`               // api路径
	Description string `json:"description,optional" form:"description,optional"` // api中文描述
	ApiGroup    string `json:"apiGroup,optional" form:"apiGroup,optional"`       // api组
	Method      string `json:"method,optional" form:"method,optional"`           // 方法:创建POST(默认)|查看GET|更新PUT|删除DELETE
	Model
}

type GetApiListRequest struct {
	OrderKey string `form:"orderKey,optional"` // 排序的字段
	Desc     bool   `form:"desc,optional"`     // 排序方式:升序false(默认) | 降序true
	SysApi
	PageRequest
}

type GetApiListResponse struct {
	List []SysApi `json:"list"`
	PageResponse
}

type CreateApiRequest struct {
	SysApi
}

type DeleteApiRequest struct {
	SysApi
}

type GetAllApiListRequest struct {
}

type GetAllApiListResponse struct {
	ApiList []SysApi `json:"apiList"`
}

type CasbinInfo struct {
	Path   string `json:"path"`   // 路径
	Method string `json:"method"` // 方法
}

type GetPathByAuthorityIdRequest struct {
	AuthorityId int64 `json:"authorityId"`
}

type GetPathByAuthorityIdResponse struct {
	List []CasbinInfo `json:"list"`
}

type GetBaseMenuTreeRequest struct {
}

type GetBaseMenuTreeResponse struct {
	SysBaseMenuList []SysBaseMenu `json:"list"`
}

type GetMenuAuthorityRequest struct {
	AuthorityId int64 `json:"authorityId"`
}

type GetMenuAuthorityResponse struct {
	SysMenuList []SysMenu `json:"list"`
}

type AddAuthorityMenuRequest struct {
	AuthorityId int64  `json:"authorityId"`
	MenuIds     string `json:"menuIds"`
}

type UpdateCasbinDataRequest struct {
	AuthorityId    int64        `json:"authorityId"`
	CasbinInfoList []CasbinInfo `json:"casbinInfoList"`
}

type UpdateAuthorityRequest struct {
	SysAuthority
}

type UpdateAuthorityResponse struct {
	SysAuthority SysAuthority `json:"sysAuthority"`
	Message      string       `json:"message"`
}

type GetBaseMenuByIdRequest struct {
	Id int64 `json:"id"`
}

type GetBaseMenuByIdResponse struct {
	SysBaseMenu SysBaseMenu `json:"sysBaseMenu"`
}

type UpdateBaseMenuRequest struct {
	SysBaseMenu
}

type CreateAuthorityRequest struct {
	SysAuthority
}

type CreateAuthorityResponse struct {
	SysAuthority SysAuthority `json:"sysAuthority"`
}

type GetUserListRequest struct {
	PageRequest
}

type GetUserListResponse struct {
	List []UserInfo `json:"list"`
	PageResponse
}

type RegisterRequest struct {
	Username     string  `json:"userName"`           // 用户名
	Password     string  `json:"passWord"`           // 密码
	NickName     string  `json:"nickName"`           // 昵称
	HeaderImg    string  `json:"headerImg,optional"` // 头像链接
	AuthorityId  int64   `json:"authorityId"`        // 角色id int64
	Enable       int64   `json:"enable,optional"`    // 是否启用
	AuthorityIds []int64 `json:"authorityIds"`       // 角色id []int64
	Phone        string  `json:"phone,optional"`     // 电话号码
	Email        string  `json:"email,optional"`     // 电子邮箱
}

type RegisterResponse struct {
	UserInfo
}

type UpdateUserInfoRequest struct {
	ID           int64   `json:"ID"`                    // 主键ID
	NickName     string  `json:"nickName,optional"`     // 用户昵称
	Phone        string  `json:"phone,optional"`        // 用户手机号
	AuthorityIds []int64 `json:"authorityIds,optional"` // 角色ID
	Email        string  `json:"email,optional"`        // 用户邮箱
	HeaderImg    string  `json:"headerImg,optional"`    // 用户头像
	SideMode     string  `json:"sideMode,optional"`     // 用户侧边主题
	Enable       int64   `json:"enable,optional"`       //冻结用户
}

type ResetUserPasswordRequest struct {
	UserID int64 `json:"userId"`
}

type DeleteUserRequest struct {
	UserID int64 `json:"userId"`
}

type DeleteApisByIdsRequest struct {
	Ids []int64 `json:"ids,optional" form:"ids,optional"`
}

type UploadFileImgRequest struct {
}

type UploadFileImgResponse struct {
	FileImgUrl string `json:"fileImgUrl"`
}

type UpdateCasbinDataByApiIdsRequest struct {
	AuthorityId int64   `json:"authorityId"`
	ApiIds      []int64 `json:"apiIds"`
}

type UpdateApiRequest struct {
	SysApi
}
