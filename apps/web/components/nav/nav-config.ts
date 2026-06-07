// 导航配置：唯一真源。顶部一级菜单 + 左侧二级菜单共用，避免两处声明同步漂移。
// 一级菜单 href = 该级默认入口，规则：有子项时指向第一个子项（点一级菜单 = 进入该模块），
// 无子项时指向自身（一级菜单本身就是一个页面，如"云网络管理"）。
export type NavLeaf = {
  label: string;
  href: string;
  // 需要的页面权限码（缺省表示无需权限，所有登录用户可见）
  // 仅 kind=page 权限码出现在这里；action 权限不影响菜单可见性
  permission?: string;
};

export type NavSection = {
  label: string;
  href: string;
  children: NavLeaf[];
  // 一级菜单的可见性：通常 children 至少一个可见则整个 section 可见
  // 显式 permission 用于无 children 的一级（如"云网络管理"）
  permission?: string;
};

// 排列规则：业务模块在前，"系统管理"始终置于一级菜单末位（系统/设置类入口靠后是常见信息架构惯例）。
// 后续新增业务一级菜单时插在"系统管理"之前。
export const NAV: NavSection[] = [
  {
    label: "设备管理",
    href: "/endpoints",
    children: [
      { label: "员工终端管理", href: "/endpoints", permission: "endpoint:read" },
      { label: "哑终端管理", href: "/dumb-terminals", permission: "dumb_terminal:read" },
      { label: "网络设备管理", href: "/network-devices", permission: "network_device:read" },
    ],
  },
  {
    label: "云网络管理",
    href: "/cloud-network",
    permission: "cloud_network:read",
    children: [],
  },
  {
    label: "系统管理",
    href: "/contacts",
    children: [
      { label: "用户管理", href: "/contacts", permission: "contact:read" },
      { label: "角色权限管理", href: "/roles", permission: "role:read" },
    ],
  },
];

// findActiveSection 按 pathname 反查激活的一级菜单。
// 优先匹配子项命中（含子路径，便于详情页 /endpoints/xxx 仍高亮父级），
// 再退化为一级 href 精确/前缀匹配，最后兜底返回第一个一级。
export function findActiveSection(pathname: string): NavSection {
  for (const section of NAV) {
    for (const child of section.children) {
      if (pathname === child.href || pathname.startsWith(child.href + "/")) {
        return section;
      }
    }
    if (pathname === section.href || pathname.startsWith(section.href + "/")) {
      return section;
    }
  }
  return NAV[0];
}

// isLeafActive 用于左侧二级菜单的高亮判定，子路径同样视为命中。
export function isLeafActive(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(href + "/");
}

// 按当前用户权限码过滤导航：
//   - children 节点：permission 命中或缺省时可见
//   - section 本身：若指定了 permission（无 children 的扁平 section）按 permission 判可见；
//     否则只要任一 child 可见，section 就可见
// admin 用户直接返回全部（已在 /me/permissions 返回全部 code，这里无需特殊处理）
export function filterNavByPermissions(perms: Set<string>): NavSection[] {
  const has = (code?: string) => !code || perms.has(code);
  return NAV.map((s) => {
    const visibleChildren = s.children.filter((c) => has(c.permission));
    return { ...s, children: visibleChildren };
  }).filter((s) => {
    if (s.children.length > 0) return true;
    return has(s.permission);
  });
}
