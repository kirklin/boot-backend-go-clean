
## 目录结构的解释：

1. `cmd/`: 应用程序的入口点。

2. `internal/`: 包含不被其他项目导入的包。
    - `domain/`: 包含核心业务逻辑。
        - `entity/`: 定义核心业务实体。
        - `repository/`: 定义仓储**接口**。
        - `usecase/`: 包含应用程序的用例（业务逻辑）。
    - `infrastructure/`: 包含与外部系统交互的具体实现。
        - `database/`: 数据库相关的实现。
        - `auth/`: 认证相关的实现。
    - `interfaces/`: 包含处理外部请求的适配器。
        - `http/`: HTTP相关的处理器、中间件和路由。

3. `pkg/`: 可以被外部项目使用的库代码。

4. `test/`: 包含测试文件，特别是模拟（mocks）。

这个结构清晰地分离了关注点：

- Domain 层包含核心业务逻辑和接口定义。
- Infrastructure 层提供了这些接口的具体实现。
- Interfaces 层处理外部请求，并使用 Domain 层的用例来处理这些请求。
