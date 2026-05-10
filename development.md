## 📋 项目开发任务书：北邮空教室查询系统 (Go 版)

### 1. 项目目标

构建一个自动化系统，每日同步北京邮电大学教务系统的空教室占用情况，并通过移动端友好的 Web 界面进行展示。

### 2. 核心技术栈

- **后端**: Go 1.20+ (推荐使用 Gin 或 Echo 框架)。
- **数据库**: MySQL 8.0。
- **前端**: Vue 3 + Vant UI (移动端组件库) + Tailwind CSS。
- **部署**: Docker + Docker-compose。

------

### 3. 核心模块规格说明

#### A. 爬虫与认证模块 (Crawler Service)

这是项目的“发动机”，必须严格复刻以下逻辑：

- **加密算法 (AES-128-ECB)**:
  - **Key**: `qzkj1kjghd=876&*`。
  - **处理流程**: 将明文密码包装成 JSON 字符串（如 `"password"`） $\rightarrow$ 执行 AES-ECB-PKCS7 加密 $\rightarrow$ 对结果进行两层 Base64 编码（对应 JS 中的 `window.btoa(Kw.encrypt(pwd))`）。
- **登录接口**: `POST [http://jwglweixin.bupt.edu.cn/bjyddx/login](http://jwglweixin.bupt.edu.cn/bjyddx/login)`。
  - **Payload**: `userNo` (学号), `pwd` (双重加密密文), `encode: "1"`。
- **数据接口**: `GET [http://jwglweixin.bupt.edu.cn/bjyddx/todayClassrooms?campusId=0](http://jwglweixin.bupt.edu.cn/bjyddx/todayClassrooms?campusId=0)` (0 为西土城，1 为沙河)。
- **必须携带的 Header**:
  - `Referer: [https://jwglweixin.bupt.edu.cn/sjd/](https://jwglweixin.bupt.edu.cn/sjd/)`。
  - `User-Agent`: 模拟手机浏览器（如 iPhone 或 Android）。

#### B. 数据持久化 (Database Design)

设计 MySQL 表 `classroom_status`：

- `id`: 自增主键。
- `building`: 教学楼名称。
- `room_number`: 教室号。
- `occupancy`: 14 位字符串（如 `01100000000000`），代表全天 14 节课的占用状态。
- `date`: 数据所属日期。

#### C. 移动端前端 (Mobile UI)

设计风格需符合手机使用习惯：

- **首页**: 各教学楼的列表卡片，显示当前空闲教室比例。
- **交互**: 下拉刷新 (Vant PullRefresh)、校区切换 (Tabs)。
- **详情视图**: 模仿电影票选座的 14 宫格（7×2），用颜色（绿/红）区分当前及后续的占用情况。

------

### 4. 开发注意事项 (Attention Points)

1. **静态加密特性**: 北邮该接口加密为静态加密。程序应支持从环境变量（`.env`）读取学号和已生成的加密密文，或实时计算密文。
2. **定时任务管理**:
   - 设定在 **每天 05:30 - 06:00** 运行。
   - **重试逻辑**: 接口不稳定时，需实现指数退避重试（间隔 5min, 10min, 15min）。
3. **安全性 (Security)**:
   - **严禁**在代码仓库中硬编码真实的个人学号和密码。
   - 配置 `.gitignore` 忽略 `.env` 文件。
4. **性能优化**:
   - 由于 Go 的并发优势，在抓取多栋楼数据时，请使用 `goroutine` 配合 `channel` 提高效率。
   - 后端 API 建议对今日数据做内存缓存或 Redis 缓存。