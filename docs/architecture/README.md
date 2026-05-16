# docs/architecture/

系统架构设计文档。

## 常见文件

```
architecture/
├── overview.md         # 系统整体架构（组件关系、数据流向）
├── tech-stack.md       # 技术选型与决策说明
└── data-model.md       # 核心数据模型 ER 图与说明
```

记录架构决策时建议使用 **ADR（Architecture Decision Record）** 格式：

```
# ADR-001: 使用 PostgreSQL 作为主数据库

## 状态
已采纳

## 背景
...

## 决策
...

## 后果
...
```
