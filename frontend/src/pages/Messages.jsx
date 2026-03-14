import React, { useState, useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { imApi, accountApi } from '../api/client';
import { motion, AnimatePresence } from 'framer-motion';
import { Send, Search, ArrowLeft } from 'lucide-react';
import { clsx } from 'clsx';

const Messages = () => {
  const { user, token } = useAuth();
  const [conversations, setConversations] = useState([]);
  const [activeConversation, setActiveConversation] = useState(null);
  const [messages, setMessages] = useState([]);
  const [inputText, setInputText] = useState('');
  const [socket, setSocket] = useState(null);
  const [searchParams] = useSearchParams();
  const targetUserId = searchParams.get('target');
  const messagesEndRef = useRef(null);

  // 1. Initialize WebSocket
  useEffect(() => {
    if (!token) return;

    const ws = new WebSocket(`ws://${window.location.host}/im/ws`); // Adjust host if needed

    ws.onopen = () => {
      console.log('WebSocket Connected');
      ws.send(JSON.stringify({ type: 'auth', token }));
    };

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'auth_ok') {
        console.log('Auth OK');
      } else if (data.content) {
        // New message received
        // Update messages if in active conversation
        if (activeConversation && (data.conversation_id === activeConversation.id || data.sender_id === activeConversation.other_user.id)) {
            setMessages((prev) => [...prev, data]);
            scrollToBottom();
        }
        // Refresh conversations list to update unread count/last message
        fetchConversations();
      }
    };

    ws.onclose = () => console.log('WebSocket Disconnected');
    
    setSocket(ws);

    return () => {
      ws.close();
    };
  }, [token, activeConversation]);

  // 2. Fetch Conversations
  const fetchConversations = async () => {
    try {
      const res = await imApi.getConversations();
      // Enrich conversations with other user info if needed
      // The API returns conversation object with user_a and user_b.
      // We need to figure out who the other user is.
      const enriched = await Promise.all(res.data.map(async (conv) => {
        const otherUserId = conv.user_a === user.id ? conv.user_b : conv.user_a;
        // Ideally we should cache user info or fetch in batch.
        // For now, fetch individually (inefficient but simple).
        try {
            const userRes = await accountApi.findByID(otherUserId);
            return { ...conv, other_user: userRes.data };
        } catch (e) {
            return { ...conv, other_user: { id: otherUserId, username: 'Unknown' } };
        }
      }));
      setConversations(enriched);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    fetchConversations();
  }, [user]);

  // 3. Handle Target User from URL (Start new chat)
  useEffect(() => {
    if (targetUserId && conversations.length > 0) {
      const existing = conversations.find(c => c.other_user.id === parseInt(targetUserId));
      if (existing) {
        setActiveConversation(existing);
      } else {
        // Create temporary conversation object or just set active user
        // Since API doesn't have "create conversation" explicit endpoint (it's implicit on send),
        // we can just set a temporary state.
        // But for simplicity, let's just fetch user info and set as active.
        accountApi.findByID(parseInt(targetUserId)).then(res => {
            setActiveConversation({
                id: null, // No conversation ID yet
                other_user: res.data,
                messages: [] 
            });
        });
      }
    }
  }, [targetUserId, conversations]);

  // 4. Fetch Messages for Active Conversation
  useEffect(() => {
    if (!activeConversation || !activeConversation.id) return;

    const fetchMessages = async () => {
      try {
        const res = await imApi.getMessages(activeConversation.id);
        setMessages(res.data.reverse()); // Assuming API returns latest first? Docs say "ListMessages", usually latest first.
        scrollToBottom();
        // Mark as read
        await imApi.markAsRead(activeConversation.id);
        fetchConversations(); // Update unread count
      } catch (err) {
        console.error(err);
      }
    };

    fetchMessages();
  }, [activeConversation]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const handleSendMessage = async (e) => {
    e.preventDefault();
    if (!inputText.trim() || !activeConversation) return;

    try {
      const res = await imApi.sendMessage(activeConversation.other_user.id, inputText);
      const newMessage = res.data;
      setMessages((prev) => [...prev, newMessage]);
      setInputText('');
      scrollToBottom();
      
      // If it was a new conversation, update ID
      if (!activeConversation.id) {
        setActiveConversation(prev => ({ ...prev, id: newMessage.conversation_id }));
        fetchConversations();
      }
    } catch (err) {
      console.error(err);
      alert(err.response?.data?.error || 'Failed to send');
    }
  };

  return (
    <div className="flex h-[calc(100vh-100px)] max-w-6xl mx-auto bg-surface border border-border rounded-xl overflow-hidden shadow-2xl">
      {/* Sidebar List */}
      <div className={clsx(
        "w-full md:w-1/3 border-r border-border flex flex-col bg-surfaceHighlight/50",
        activeConversation ? "hidden md:flex" : "flex"
      )}>
        <div className="p-4 border-b border-border">
          <h2 className="text-xl font-bold mb-4">Messages</h2>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-textSecondary" size={18} />
            <input
              type="text"
              placeholder="Search..."
              className="w-full pl-10 pr-4 py-2 bg-background rounded-lg text-sm focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
        </div>
        
        <div className="flex-1 overflow-y-auto">
          {conversations.map((conv) => (
            <div
              key={conv.id}
              onClick={() => setActiveConversation(conv)}
              className={clsx(
                "p-4 border-b border-border cursor-pointer hover:bg-surfaceHighlight transition-colors",
                activeConversation?.id === conv.id ? "bg-surfaceHighlight" : ""
              )}
            >
              <div className="flex justify-between items-start mb-1">
                <h3 className="font-bold">{conv.other_user.username}</h3>
                <span className="text-xs text-textSecondary">
                  {new Date(conv.last_message_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <p className="text-sm text-textSecondary truncate max-w-[180px]">
                  {/* Last message content not in conversation object? Docs say "last_message_at". 
                      Maybe we need to fetch last message separately or it's not shown.
                      I'll just show "Click to view" or nothing.
                  */}
                  Click to view
                </p>
                {conv.unread_count > 0 && (
                  <span className="bg-accent text-black text-xs font-bold px-2 py-0.5 rounded-full">
                    {conv.unread_count}
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Chat Area */}
      <div className={clsx(
        "w-full md:w-2/3 flex flex-col bg-black",
        !activeConversation ? "hidden md:flex" : "flex"
      )}>
        {activeConversation ? (
          <>
            <div className="p-4 border-b border-border flex items-center gap-4 bg-surface">
              <button 
                onClick={() => setActiveConversation(null)}
                className="md:hidden p-2 hover:bg-surfaceHighlight rounded-full"
              >
                <ArrowLeft size={20} />
              </button>
              <h3 className="text-lg font-bold">{activeConversation.other_user.username}</h3>
            </div>

            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {messages.map((msg) => {
                const isMe = msg.sender_id === user.id;
                return (
                  <div
                    key={msg.id}
                    className={clsx(
                      "flex",
                      isMe ? "justify-end" : "justify-start"
                    )}
                  >
                    <div
                      className={clsx(
                        "max-w-[70%] px-4 py-2 rounded-2xl text-sm",
                        isMe
                          ? "bg-white text-black rounded-tr-none"
                          : "bg-surfaceHighlight text-white rounded-tl-none"
                      )}
                    >
                      {msg.content}
                    </div>
                  </div>
                );
              })}
              <div ref={messagesEndRef} />
            </div>

            <form onSubmit={handleSendMessage} className="p-4 border-t border-border bg-surface flex gap-4">
              <input
                type="text"
                value={inputText}
                onChange={(e) => setInputText(e.target.value)}
                placeholder="Type a message..."
                className="flex-1 px-4 py-2 bg-background rounded-full focus:outline-none focus:ring-1 focus:ring-accent"
              />
              <button
                type="submit"
                disabled={!inputText.trim()}
                className="p-2 bg-white text-black rounded-full hover:bg-gray-200 disabled:opacity-50 transition-colors"
              >
                <Send size={20} />
              </button>
            </form>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-textSecondary">
            Select a conversation to start chatting
          </div>
        )}
      </div>
    </div>
  );
};

export default Messages;
