# API Contract

## 目标

本文件定义首版正式 API 协议。前端、人类开发者和 AI agent 都必须按此实现，不得各自发明字段。

基础约束：
- Base path: `/api/v1`
- Content-Type: `application/json`
- 导出接口除外，均返回 JSON

## Shared Enums

### Job Status

- `queued`
- `running`
- `succeeded`
- `failed`

### Pipeline Stage

- `ingest`
- `outline`
- `entities`
- `scene_planning`
- `screenplay_generation`
- `validation`
- `persistence`

### Generation Mode

- `deterministic`
- `llm`

## Shared Response Shape

成功响应：

```json
{
  "data": {},
  "error": null,
  "meta": {
    "request_id": "req_123"
  }
}
```

失败响应：

```json
{
  "data": null,
  "error": {
    "code": "invalid_input",
    "message": "at least 3 chapters are required",
    "details": {}
  },
  "meta": {
    "request_id": "req_123"
  }
}
```

错误码首版至少支持：
- `invalid_input`
- `job_not_found`
- `job_not_ready`
- `generation_failed`
- `internal_error`

## POST `/jobs`

创建生成任务。

Request:

```json
{
  "source": {
    "title": "夜雨疑云",
    "author": "示例作者",
    "chapters": [
      {
        "index": 1,
        "title": "第一章",
        "content": "..."
      },
      {
        "index": 2,
        "title": "第二章",
        "content": "..."
      },
      {
        "index": 3,
        "title": "第三章",
        "content": "..."
      }
    ]
  },
  "adaptation": {
    "style": "悬疑网剧",
    "audience": "大众向",
    "notes": ["强化冲突"]
  },
  "generation": {
    "mode": "deterministic"
  }
}
```

Request constraints:
- `source.title`: required
- `source.chapters`: required, length >= 3
- `source.chapters[].index`: required, continuous from 1
- `source.chapters[].title`: required
- `source.chapters[].content`: required
- `adaptation.style`: required
- `generation.mode`: required

Response `202 Accepted`:

```json
{
  "data": {
    "job": {
      "id": "job_123",
      "status": "queued",
      "current_stage": "ingest",
      "progress_percent": 0,
      "created_at": "2026-06-05T01:00:00Z"
    }
  },
  "error": null,
  "meta": {
    "request_id": "req_123"
  }
}
```

## GET `/jobs/{job_id}`

查询任务状态。

Response `200 OK`:

```json
{
  "data": {
    "job": {
      "id": "job_123",
      "status": "running",
      "current_stage": "scene_planning",
      "progress_percent": 62,
      "source_title": "夜雨疑云",
      "generation_mode": "deterministic",
      "warnings": [],
      "error_message": "",
      "created_at": "2026-06-05T01:00:00Z",
      "updated_at": "2026-06-05T01:01:20Z"
    },
    "stages": [
      {
        "name": "ingest",
        "status": "succeeded"
      },
      {
        "name": "outline",
        "status": "succeeded"
      },
      {
        "name": "entities",
        "status": "succeeded"
      },
      {
        "name": "scene_planning",
        "status": "running"
      }
    ]
  },
  "error": null,
  "meta": {
    "request_id": "req_123"
  }
}
```

## GET `/jobs/{job_id}/result`

获取结构化结果。仅当 job 状态为 `succeeded` 时返回成功。

Response `200 OK`:

```json
{
  "data": {
    "job_id": "job_123",
    "screenplay": {
      "version": "1.0",
      "source": {},
      "adaptation": {},
      "characters": [],
      "locations": [],
      "scenes": [],
      "validation": {
        "status": "passed",
        "warnings": []
      }
    },
    "yaml_text": "version: \"1.0\"\n..."
  },
  "error": null,
  "meta": {
    "request_id": "req_123"
  }
}
```

## GET `/jobs/{job_id}/export`

下载 YAML 文件。

Response `200 OK`:
- `Content-Type: application/x-yaml`
- `Content-Disposition: attachment; filename="<job_id>.screenplay.yaml"`

Body:
- raw YAML text

## Optional Next Endpoint

如果时间允许，可增加：
- `POST /jobs/{job_id}/retry`

但这不是阶段 1 的 blocker。

## Non-Negotiable Rules

- 前端不得依赖 undocumented 字段
- 后端不得擅自改动枚举值
- 返回结构必须带 `meta.request_id`
- `GET /jobs/{job_id}/result` 必须返回 `screenplay` 和 `yaml_text`
