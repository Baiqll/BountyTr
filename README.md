# BountyTr

赏金目标追踪工具，自动获取 HackerOne、Bugcrowd、Intigriti 平台的赏金项目及其 scope 域名。

```
         __                      __        __      
        / /_  ____  __  ______  / /___  __/ /______
       / __ \/ __ \/ / / / __ \/ __/ / / / __/ ___/
      / /_/ / /_/ / /_/ / / / / /_/ /_/ / /_/ /    
     /_.___/\____/\__,_/_/ /_/\__/\__, /\__/_/     
                                 /____/       v3.0       
```

## 特性

- 支持 HackerOne、Bugcrowd、Intigriti 三大平台
- GitHub 数据 + API 接口混合模式，快速获取最新项目
- HackerOne 私有项目支持（需配置 API Token）
- 智能缓存机制，避免重复请求
- 增量更新，只输出新增域名
- 定时监控模式
- 钉钉通知支持

## 安装

```bash
go install github.com/baiqll/bountytr/cmd/bountytr@latest
```

或从源码编译：

```bash
git clone https://github.com/baiqll/bountytr.git
cd bountytr
go build ./cmd/bountytr/
```

## 使用

```bash
# 基本使用 - 获取所有平台数据
bountytr

# 静默模式 - 只输出新增域名
bountytr -silent

# 定时监控 - 每60分钟执行一次
bountytr -t 60

# 指定项目 - 获取特定项目的 scope
bountytr -handle <project_name>
```

### 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-t` | 监控周期（分钟），0 表示只运行一次 | 0 |
| `-silent` | 静默模式，不显示状态信息 | false |
| `-handle` | 指定项目名称 | - |

## 配置

配置文件位置：`~/.config/bountytr/config.yaml`

首次运行会自动生成默认配置：

```yaml
HackerOne:
  Enable: true
  Private:
    Enable: false
    ApiName: ""
    ApiToken: ""

Bugcrowd:
  Enable: true
  Private:
    Enable: false

Intigriti:
  Enable: true
  Private:
    Enable: false

Black:
  - ".gov"
  - ".edu"
  - ".json"
  - ".[0-9.]+$"
  - "github.com/"

DingTalk:
  AppKey: ""
  AppSecret: ""
```

### HackerOne 私有项目配置

1. 登录 HackerOne，进入 Settings -> API Token
2. 创建 API Token，获取 `ApiName` 和 `ApiToken`
3. 配置文件中设置：

```yaml
HackerOne:
  Enable: true
  Private:
    Enable: true
    ApiName: "your_api_name"
    ApiToken: "your_api_token"
```

## 输出文件

所有输出文件保存在 `~/.config/bountytr/` 目录：

| 文件 | 说明 |
|------|------|
| `domain.txt` | 所有已发现的域名（增量） |
| `bugbounty-public.txt` | 公开项目 URL 列表 |
| `bugbounty-private.txt` | 私有项目 URL 列表 |
| `faildomain.txt` | 匹配失败的目标 |
| `cache/` | 缓存目录 |

### API 新增标记

`bugbounty-public.txt` 中，API 获取到但 GitHub 数据中没有的项目会用 `== API NEW ==` 标记：

```
https://hackerone.com/project1
https://hackerone.com/project2
== API NEW ==
https://hackerone.com/new_project
```

## 数据来源

- 公开项目：[bounty-targets-data](https://github.com/arkadiyt/bounty-targets-data)（GitHub 缓存，每小时更新）
- 私有项目：HackerOne API（需配置 Token）

## 工作流程

1. 从 GitHub 获取公开项目数据（带缓存）
2. 如果启用私有项目，通过 API 获取私有项目列表
3. 对比 GitHub 数据，标记 API 新增的公开项目
4. 提取所有项目的 scope 域名
5. 与本地已有数据对比，输出新增域名
6. 保存到对应文件

## 参考

- [bounty-targets-data](https://github.com/arkadiyt/bounty-targets-data)
- [HackerOne API](https://api.hackerone.com/)
