package authz

// PermissionDef represents a built-in permission definition used for sync.
type PermissionDef struct {
    Code        string
    Name        string
    Description *string
}

func s(s string) *string { return &s }

// BuiltinPermissions is the canonical list of permissions defined in code.
// SyncFromCode will upsert these into the database.
var BuiltinPermissions = []PermissionDef{
    // System management
    {Code: "system.role.manage", Name: "角色管理", Description: s("管理角色及其权限")},
    {Code: "system.user.manage", Name: "用户管理", Description: s("管理用户及其角色")},
    {Code: "system.permission.manage", Name: "权限管理", Description: s("管理权限定义与同步")},

    // Traffic monitor
    {Code: "traffic.read", Name: "流量监控查看", Description: s("查看流量监控面板")},

    // School management
    {Code: "school.manage", Name: "学校管理", Description: s("管理学校及其信息")},

    // Settlement
    {Code: "settlement.read", Name: "结算查看", Description: s("查看结算数据与报表")},
    {Code: "settlement.calculate", Name: "结算计算", Description: s("创建/删除结算任务，更新结算配置")},

    // Rates (under settlement)
    {Code: "rates.customer.read", Name: "客户业务费率查看", Description: s("查看客户业务费率")},
    {Code: "rates.customer.write", Name: "客户业务费率维护", Description: s("新增/修改客户业务费率")},
    {Code: "rates.node.read", Name: "节点业务费率查看", Description: s("查看节点业务费率")},
    {Code: "rates.node.write", Name: "节点业务费率维护", Description: s("新增/修改节点业务费率")},
    {Code: "rates.final.read", Name: "最终客户费率查看", Description: s("查看最终客户费率")},
    {Code: "rates.final.write", Name: "最终客户费率维护", Description: s("新增/修改/刷新最终客户费率")},

    // Business entities (under settlement)
    {Code: "entities.read", Name: "业务对象查看", Description: s("查看业务对象")},
    {Code: "entities.write", Name: "业务对象维护", Description: s("新增/修改/删除业务对象")},

    // Operation logs
    {Code: "operation_logs.read", Name: "操作日志查看", Description: s("查询与导出操作日志")},
}
