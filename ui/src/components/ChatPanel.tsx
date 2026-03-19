import { useState, useEffect, useRef, useCallback } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
import { Message } from '../types';

const ChatPanel: React.FC = () => {
  const { status: wsStatus, lastMessage, sendMessage } = useWebSocket(true);
  const [messages, setMessages] = useState<Message[]>([]);
  const [message, setMessage] = useState('');
  const [sending, setSending] = useState(false);
  const [sessionId, setSessionId] = useState<string>('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const pendingRequests = useRef<Map<string, { resolve: (value: unknown) => void; reject: (reason: unknown) => void }>>(new Map());

  // 监听连接建立，获取session_id
  useEffect(() => {
    if (lastMessage?.method === 'connected') {
      const params = lastMessage.params as { session_id: string };
      setSessionId(params.session_id);
      console.log('WebSocket connected with session:', params.session_id);
    }
  }, [lastMessage]);

  // 处理消息响应
  useEffect(() => {
    if (lastMessage?.id && pendingRequests.current.has(lastMessage.id)) {
      const pending = pendingRequests.current.get(lastMessage.id);
      if (pending) {
        if (lastMessage.error) {
          pending.reject(new Error(lastMessage.error.message));
        } else {
          pending.resolve(lastMessage.result);
        }
        pendingRequests.current.delete(lastMessage.id);
      }
    }

    // 监听来自gateway的消息
    // 后端 BroadcastNotification 会把消息包装在 params.data 中
    if (lastMessage?.method === 'chat.response') {
      console.log('Received chat.response:', lastMessage);
      const params = lastMessage.params as { data?: { content: string }; content?: string };
      // 兼容两种格式：params.data.content 和 params.content
      const content = params.data?.content || params.content;
      if (content) {
        const assistantMessage: Message = {
          id: Date.now().toString(),
          role: 'assistant',
          content: content,
          timestamp: Date.now() / 1000,
        };
        setMessages((prev) => [...prev, assistantMessage]);
      }
    }
  }, [lastMessage]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // 发送消息并等待响应
  const sendAndWait = useCallback((method: string, params?: Record<string, unknown>): Promise<unknown> => {
    return new Promise((resolve, reject) => {
      const id = Math.random().toString(36).substring(2, 15);
      pendingRequests.current.set(id, { resolve, reject });
      sendMessage(method, params, id);

      // 设置超时
      setTimeout(() => {
        if (pendingRequests.current.has(id)) {
          pendingRequests.current.delete(id);
          reject(new Error('Request timeout'));
        }
      }, 30000);
    });
  }, [sendMessage]);

  const handleSend = async () => {
    if (!message.trim()) return;

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: message,
      timestamp: Date.now() / 1000,
    };

    setMessages((prev) => [...prev, userMessage]);
    setSending(true);

    try {
      // 直接通过WebSocket发送chat消息
      const result = await sendAndWait('chat', { content: message }) as { response?: string };
      if (result?.response) {
        const assistantMessage: Message = {
          id: (Date.now() + 1).toString(),
          role: 'assistant',
          content: result.response,
          timestamp: Date.now() / 1000,
        };
        setMessages((prev) => [...prev, assistantMessage]);
      }
    } catch (err) {
      console.error('Failed to send message:', err);
    } finally {
      setSending(false);
      setMessage('');
    }
  };

  const clearChat = () => {
    setMessages([]);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Chat</h1>
        <div className="flex items-center gap-2">
          <span className={`w-2 h-2 rounded-full ${wsStatus === 'connected' ? 'bg-green-500' : 'bg-red-500'}`} />
          <span className="text-sm text-gray-500">{wsStatus}</span>
          {sessionId && (
            <span className="text-xs text-gray-400 ml-2">
              Session: {sessionId.substring(0, 8)}...
            </span>
          )}
        </div>
      </div>

      {/* Messages */}
      <div className="bg-white rounded-lg border border-gray-200 h-[500px] flex flex-col">
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {messages.length === 0 ? (
            <div className="h-full flex items-center justify-center text-gray-400">
              <div className="text-center">
                <p>直接与 Gateway 对话</p>
                <p className="text-sm mt-2">消息通过 WebSocket 实时传输</p>
              </div>
            </div>
          ) : (
            messages.map((msg) => (
              <div
                key={msg.id}
                className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-[70%] p-3 rounded-lg ${
                    msg.role === 'user'
                      ? 'bg-blue-500 text-white'
                      : 'bg-gray-100 text-gray-800'
                  }`}
                >
                  <div className={`text-xs mb-1 capitalize ${msg.role === 'user' ? 'text-blue-200' : 'text-gray-500'}`}>
                    {msg.role}
                  </div>
                  <p className="whitespace-pre-wrap">{msg.content}</p>
                </div>
              </div>
            ))
          )}
          <div ref={messagesEndRef} />
        </div>

        {/* Input */}
        <div className="p-4 border-t border-gray-200">
          <div className="flex gap-2">
            <input
              type="text"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && !e.shiftKey && handleSend()}
              placeholder="输入消息..."
              disabled={sending || wsStatus !== 'connected'}
              className="flex-1 px-3 py-2 bg-gray-50 border border-gray-300 rounded-lg text-gray-900 disabled:opacity-50"
            />
            <button
              onClick={handleSend}
              disabled={sending || !message.trim() || wsStatus !== 'connected'}
              className="px-4 py-2 bg-blue-500 hover:bg-blue-600 disabled:bg-gray-400 rounded-lg text-white"
            >
              {sending ? '发送中...' : '发送'}
            </button>
            <button
              onClick={clearChat}
              className="px-4 py-2 bg-gray-200 hover:bg-gray-300 rounded-lg text-gray-700"
            >
              清空
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ChatPanel;