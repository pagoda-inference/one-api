# 批量处理

### 批量推理

**POST** `/v1/batches`

提交离线批量推理任务。

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| model | string | ✓ | 模型名称 |
| input_file_id | string | ✓ | 输入文件 ID |
| endpoint | string | ✓ | 目标端点 (如 `/v1/chat/completions`) |
| completion_window | string | ✓ | 完成时间窗口 (如 `24h`) |

**GET** `/v1/batches`

查询批量任务列表。

**GET** `/v1/batches/:id`

查询指定批量任务状态和结果。

**POST** `/v1/batches/:id/cancel`

取消运行中的批量任务。