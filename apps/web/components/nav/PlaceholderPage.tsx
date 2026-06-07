// 占位页：用于尚未实现的菜单项。统一展示样式，与 contacts/endpoints 页的 h-full + flex-col 结构对齐。
export function PlaceholderPage({
  title,
  description,
}: {
  title: string;
  description?: string;
}) {
  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold">{title}</h1>
      </div>
      <div className="flex min-h-0 flex-1 items-center justify-center rounded-xl border border-dashed border-zinc-300 bg-white text-sm text-zinc-400 dark:border-zinc-700 dark:bg-zinc-900">
        {description ?? "敬请期待"}
      </div>
    </div>
  );
}
