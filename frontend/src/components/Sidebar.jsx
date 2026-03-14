import React from 'react';
import { NavLink } from 'react-router-dom';
import { Home, User, MessageSquare, PlusSquare, Search, LogOut } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { clsx } from 'clsx';
import { motion } from 'framer-motion';

const Sidebar = () => {
  const { logout } = useAuth();

  const links = [
    { name: '首页', path: '/', icon: Home },
    { name: '发现', path: '/search', icon: Search },
    { name: '发布', path: '/upload', icon: PlusSquare },
    { name: '消息', path: '/messages', icon: MessageSquare },
    { name: '我的', path: '/profile', icon: User },
  ];

  return (
    <aside className="w-64 h-full bg-surface border-r border-border flex flex-col justify-between p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-wider mb-10 text-center">KAIROS</h1>
        <nav className="flex flex-col gap-4">
          {links.map((link) => (
            <NavLink
              key={link.path}
              to={link.path}
              className={({ isActive }) =>
                clsx(
                  "flex items-center gap-4 px-4 py-3 rounded-lg transition-colors duration-300",
                  isActive
                    ? "bg-surfaceHighlight text-accent shadow-[0_0_15px_rgba(255,255,255,0.1)]"
                    : "text-textSecondary hover:bg-surfaceHighlight hover:text-primary"
                )
              }
            >
              <link.icon size={24} />
              <span className="text-lg font-medium">{link.name}</span>
            </NavLink>
          ))}
        </nav>
      </div>

      <button
        onClick={logout}
        className="flex items-center gap-4 px-4 py-3 text-textSecondary hover:text-red-500 transition-colors duration-300 mt-auto"
      >
        <LogOut size={24} />
        <span className="text-lg font-medium">退出登录</span>
      </button>
    </aside>
  );
};

export default Sidebar;
