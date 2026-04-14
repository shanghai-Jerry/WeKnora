# Python Scripts

POST /api/v1/sessions
先创建 session，返回session_id

然后使用session_id， 和agent进行对话

/api/v1/agent-chat/{4aa43c71-bc80-40dd-857f-48bfa35a78e1}

LLM回复方式： 使用EventStream

type: message | Data: xxx

{"id":"LIpMwRI9SRjc","response_type":"thinking","content":" How","done":false,"data":{"event_id":"133c3934-thinking"}}	
{"id":"LIpMwRI9SRjc","response_type":"thinking","content":" can","done":false,"data":{"event_id":"133c3934-thinking"}}	
{"id":"LIpMwRI9SRjc","response_type":"thinking","content":" I","done":false,"data":{"event_id":"133c3934-thinking"}}	
{"id":"LIpMwRI9SRjc","response_type":"thinking","content":" help","done":false,"data":{"event_id":"133c3934-thinking"}}	
message {"id":"LIpMwRI9SRjc","response_type":"thinking","content":" you","done":false,"data":{"event_id":"133c3934-thinking"}}	
message {"id":"LIpMwRI9SRjc","response_type":"thinking","content":" today","done":false,"data":{"event_id":"133c3934-thinking"}}	
message {"id":"LIpMwRI9SRjc","response_type":"thinking","content":"?","done":false,"data":{"event_id":"133c3934-thinking"}}	
message {"id":"LIpMwRI9SRjc","response_type":"answer","content":"Hello! I'm WeKnora, your concise retrieval assistant. How can I help you today?","done":false,"data":{"event_id":"ca0c63fb-answer"}}	
message	{"id":"LIpMwRI9SRjc","response_type":"answer","content":"","done":true,"data":{"completed_at":1775097228,"duration_ms":1,"event_id":"ca0c63fb-answer"}}	
{"id":"LIpMwRI9SRjc","response_type":"answer","content":"","done":true,"data":{"completed_at":1775097228,"duration_ms":1,"event_id":"ca0c63fb-answer"}}	
