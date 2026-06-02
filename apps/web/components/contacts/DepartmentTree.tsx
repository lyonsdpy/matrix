"use client";

import { useEffect, useState } from "react";
import type { DepartmentNode } from "@/lib/types";

// 部门树懒加载：根节点首屏加载，展开时再请求子部门。
// 与一次性加载全树相比，2210 部门首屏只下 32 个根节点，交互更轻。
export function DepartmentTree({
  onSelect,
  selectedId,
}: {
  onSelect: (deptID: string, name: string) => void;
  selectedId: string;
}) {
  const [roots, setRoots] = useState<DepartmentNode[] | null>(null);

  useEffect(() => {
    fetch("/api/proxy/contacts/tree")
      .then((r) => r.json())
      .then((d) => setRoots(d.departments ?? []))
      .catch(() => setRoots([]));
  }, []);

  if (roots === null) {
    return <div className="px-3 py-4 text-xs text-zinc-400">加载部门树...</div>;
  }
  if (roots.length === 0) {
    return <div className="px-3 py-4 text-xs text-zinc-400">没有部门数据</div>;
  }
  return (
    <div className="text-sm">
      {roots.map((n) => (
        <TreeNode
          key={n.department_id}
          node={n}
          level={0}
          selectedId={selectedId}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
}

function TreeNode({
  node,
  level,
  selectedId,
  onSelect,
}: {
  node: DepartmentNode;
  level: number;
  selectedId: string;
  onSelect: (id: string, name: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const [children, setChildren] = useState<DepartmentNode[] | null>(null);
  const [loading, setLoading] = useState(false);

  const toggle = async () => {
    if (!node.has_children) return;
    if (!expanded && children === null) {
      setLoading(true);
      try {
        const d = await fetch(
          `/api/proxy/contacts/tree?parent=${encodeURIComponent(node.department_id)}`,
        ).then((r) => r.json());
        setChildren(d.departments ?? []);
      } catch {
        setChildren([]);
      } finally {
        setLoading(false);
      }
    }
    setExpanded((v) => !v);
  };

  const isSelected = selectedId === node.department_id;
  return (
    <div>
      <div
        style={{ paddingLeft: 8 + level * 14 }}
        className={`group flex items-center gap-1 py-1.5 pr-2 cursor-pointer ${
          isSelected
            ? "bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
            : "hover:bg-zinc-100 dark:hover:bg-zinc-800"
        }`}
      >
        <button
          type="button"
          onClick={toggle}
          className={`w-4 shrink-0 text-xs ${node.has_children ? "text-zinc-500" : "text-transparent"}`}
        >
          {node.has_children ? (expanded ? "▼" : "▶") : "·"}
        </button>
        <span
          className="flex-1 truncate"
          onClick={() => onSelect(node.department_id, node.name)}
          title={node.name}
        >
          {node.name}
        </span>
        <span className="text-xs text-zinc-400">{node.member_count}</span>
      </div>
      {loading && (
        <div
          style={{ paddingLeft: 8 + (level + 1) * 14 }}
          className="py-1 text-xs text-zinc-400"
        >
          加载中...
        </div>
      )}
      {expanded &&
        children?.map((c) => (
          <TreeNode
            key={c.department_id}
            node={c}
            level={level + 1}
            selectedId={selectedId}
            onSelect={onSelect}
          />
        ))}
    </div>
  );
}
