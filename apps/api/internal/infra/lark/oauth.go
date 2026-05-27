package lark

import (
	"context"
	"fmt"
	"net/url"

	larkauthen "github.com/larksuite/oapi-sdk-go/v3/service/authen/v1"
)

const larkAuthorizeURL = "https://open.feishu.cn/open-apis/authen/v1/authorize"

// OAuthUser 飞书 OAuth 认证后返回的用户信息。
type OAuthUser struct {
	OpenID string // 用户在当前应用内的唯一标识，跨设备一致
	UserID string // 租户内 user_id（飞书通讯录 ID）
	Name   string
	Email  string
}

// OAuthProvider 封装飞书扫码登录所需的 OAuth 操作。
type OAuthProvider struct {
	client      *Client
	appID       string
	redirectURI string
}

// NewOAuthProvider 创建 OAuthProvider，appID 和 redirectURI 从配置注入。
func NewOAuthProvider(client *Client, appID, redirectURI string) *OAuthProvider {
	return &OAuthProvider{client: client, appID: appID, redirectURI: redirectURI}
}

// AuthorizeURL 构造飞书 OAuth 授权页 URL，state 用于 CSRF 防护。
// 返回的 URL 由 Next.js 服务端生成后，浏览器直接跳转。
func (p *OAuthProvider) AuthorizeURL(state string) string {
	params := url.Values{}
	params.Set("app_id", p.appID)
	params.Set("redirect_uri", p.redirectURI)
	params.Set("state", state)
	return larkAuthorizeURL + "?" + params.Encode()
}

// ExchangeCode 用飞书返回的 code 换取用户信息，一步完成（access_token 响应直接含用户字段）。
// 使用旧版 authen/v1/access_token 接口，响应中直接包含 open_id、name 等，无需二次调用。
func (p *OAuthProvider) ExchangeCode(ctx context.Context, code string) (*OAuthUser, error) {
	grantType := "authorization_code"
	req := larkauthen.NewCreateAccessTokenReqBuilder().
		Body(larkauthen.NewCreateAccessTokenReqBodyBuilder().
			GrantType(grantType).
			Code(code).
			Build()).
		Build()

	resp, err := p.client.Authen.V1.AccessToken.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("lark oauth: exchange code: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("lark oauth: exchange code: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("lark oauth: exchange code: empty response")
	}

	user := &OAuthUser{}
	if resp.Data.OpenId != nil {
		user.OpenID = *resp.Data.OpenId
	}
	if resp.Data.UserId != nil {
		user.UserID = *resp.Data.UserId
	}
	if resp.Data.Name != nil {
		user.Name = *resp.Data.Name
	}
	if resp.Data.Email != nil {
		user.Email = *resp.Data.Email
	}
	if user.OpenID == "" {
		return nil, fmt.Errorf("lark oauth: exchange code: missing open_id in response")
	}
	return user, nil
}
