# 文件解析

**POST** `/v1/files/file_parse`

上传文件并解析其内容，支持 PDF、Word、Excel、TXT 等格式。

| 参数 | 类型 | 必填 | 说明 |
|:-----|:-----|:----:|:-----|
| file | file | ✓ | 要解析的文件 |
| model | string | ✓ | 解析模型名称 |

```bash
curl -X POST https://baotaai.bedicloud.net/v1/files/file_parse \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -F "file=@/path/to/document.pdf" \
  -F "model=bedi/document-parse"
```

```json
{
  "id": "file-xxxxx",
  "object": "file_parse",
  "status": "completed",
  "content": "这是解析后的文档内容...",
  "usage": {"prompt_tokens": 1000, "total_tokens": 1000}
}
```