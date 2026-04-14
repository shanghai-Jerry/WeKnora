#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
mitmproxy 脚本：拦截并处理 text/event-stream 数据
保存为 eventstream_monitor.py
启动命令：mitmdump -p 8888 -s eventstream_monitor.py
"""

import json
from datetime import datetime
from mitmproxy import http


class EventStreamInterceptor:
    """拦截并处理 EventStream (SSE) 数据的插件"""

    def __init__(self):
        print(f"Init EventStreamInterceptor")
        # 保存数据的文件
        self.output_file = "eventstream_log.txt"
        # 可选：按 URL 过滤，只拦截特定域名或路径的 eventstream
        self.target_hosts = ["localhost", "127.0.0.1"]  # 按需修改
        self.target_paths = ["/events", "/stream", "/sse", "/api/langgraph"]  # 按需修改

    def _should_intercept(self, flow: http.HTTPFlow) -> bool:
        """判断是否需要拦截此请求"""
        # 检查响应头是否为 event-stream
        if flow.response and flow.response.headers.get("Content-Type", ""):
            if "text/event-stream" in flow.response.headers["Content-Type"]:
                return True

        # 可选：按 URL 过滤
        # if any(host in flow.request.host for host in self.target_hosts):
        #     return True
        # if any(path in flow.request.path for path in self.target_paths):
        #     return True

        return False

    def response(self, flow: http.HTTPFlow):
        """拦截响应"""
        if not self._should_intercept(flow):
            print(f"Skip non-Stream response: {flow.request.pretty_url}")
            return

        print(f"\n{'='*60}")
        print(f"[{datetime.now().strftime('%Y-%m-%d %H:%M:%S')}] 检测到 EventStream 响应")
        print(f"URL: {flow.request.pretty_url}")
        print(f"Response Status: {flow.response.status_code}")

        # 获取响应内容
        content_type = flow.response.headers.get("content-type", "unknown")
        print(f"Content-Type: {content_type}")

        # 获取响应体
        if flow.response.raw_content:
            body_text = flow.response.get_text()
            print(f"响应内容长度: {len(body_text)} 字符")

            # 解析 event-stream 格式数据
            self._parse_sse_data(body_text)

            # 可选：将原始数据保存到文件
            self._save_to_file(flow, body_text)

    def _parse_sse_data(self, data: str):
        """解析 Server-Sent Events 格式的数据"""
        lines = data.split('\n')
        event_data = {}

        for line in lines:
            line = line.rstrip('\r')
            if not line:
                # 空行表示一个事件结束
                if event_data:
                    print(f"  SSE 事件: {event_data}")
                    event_data = {}
                continue

            if line.startswith('data:'):
                event_data['data'] = line[5:].lstrip()
            elif line.startswith('event:'):
                event_data['event_type'] = line[6:].lstrip()
            elif line.startswith('id:'):
                event_data['id'] = line[3:].lstrip()
            elif line.startswith('retry:'):
                event_data['retry'] = line[6:].lstrip()

        # 处理最后可能未以空行结束的数据
        if event_data:
            print(f"  SSE 事件: {event_data}")

    def _save_to_file(self, flow: http.HTTPFlow, data: str):
        """将拦截到的数据保存到文件"""
        with open(self.output_file, 'a', encoding='utf-8') as f:
            f.write(f"\n{'='*60}\n")
            f.write(f"Timestamp: {datetime.now().isoformat()}\n")
            f.write(f"URL: {flow.request.pretty_url}\n")
            f.write(f"Method: {flow.request.method}\n")
            f.write(f"Request Headers: {dict(flow.request.headers)}\n")
            f.write(f"Response Headers: {dict(flow.response.headers)}\n")
            f.write(f"Response Body:\n{data}\n")


# 插件入口：mitmproxy 会查找名为 addons 的列表
addons = [EventStreamInterceptor()]