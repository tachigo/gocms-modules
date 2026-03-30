package permission

import "testing"

func TestNormalizeModule(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// 复数 → 单数
		{"articles", "article"},
		{"pages", "page"},
		{"users", "user"},
		{"menus", "menu"},
		{"taxonomies", "taxonomy"},
		// 连字符路径 → 模块名
		{"menu-groups", "menu"},
		// 不同路径名 → 模块名
		{"roles", "permission"},
		{"permissions", "permission"},
		// 路径与模块名一致，直接透传
		{"media", "media"},
		{"settings", "settings"},
		// 未知路径段，原样返回
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeModule(tt.input)
			if got != tt.want {
				t.Errorf("normalizeModule(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseRequestPath(t *testing.T) {
	tests := []struct {
		path       string
		method     string
		wantModule string
		wantAction string
	}{
		// 文章列表
		{"/api/admin/articles", "GET", "article", "read"},
		// 文章详情
		{"/api/admin/articles/123", "GET", "article", "read"},
		// 创建文章
		{"/api/admin/articles", "POST", "article", "create"},
		// 更新文章
		{"/api/admin/articles/123", "PUT", "article", "update"},
		// 删除文章
		{"/api/admin/articles/123", "DELETE", "article", "delete"},
		// 分类管理
		{"/api/admin/taxonomies", "GET", "taxonomy", "read"},
		{"/api/admin/taxonomies/1/terms", "GET", "taxonomy", "read"},
		// 菜单分组
		{"/api/admin/menu-groups", "GET", "menu", "read"},
		{"/api/admin/menus", "POST", "menu", "create"},
		// 角色管理 → permission 模块
		{"/api/admin/roles", "GET", "permission", "read"},
		{"/api/admin/roles", "POST", "permission", "create"},
		// 页面
		{"/api/admin/pages", "GET", "page", "read"},
		// 用户
		{"/api/admin/users", "GET", "user", "read"},
		// media（无需映射）
		{"/api/admin/media", "GET", "media", "read"},
		// settings（无需映射）
		{"/api/admin/settings", "GET", "settings", "read"},
		// 非 admin 路径
		{"/api/articles", "GET", "", ""},
		// PATCH 方法
		{"/api/admin/articles/1", "PATCH", "article", "update"},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			gotMod, gotAct := parseRequestPath(tt.path, tt.method)
			if gotMod != tt.wantModule || gotAct != tt.wantAction {
				t.Errorf("parseRequestPath(%q, %q) = (%q, %q), want (%q, %q)",
					tt.path, tt.method, gotMod, gotAct, tt.wantModule, tt.wantAction)
			}
		})
	}
}
