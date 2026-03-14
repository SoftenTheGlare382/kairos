import React from 'react';
import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import { motion } from 'framer-motion';

const Layout = () => {
  return (
    <div className="flex h-screen bg-background text-textPrimary font-sans overflow-hidden">
      <Sidebar />
      <motion.main
        className="flex-1 overflow-y-auto p-8 relative scrollbar-hide"
        initial={{ opacity: 0, x: 20 }}
        animate={{ opacity: 1, x: 0 }}
        exit={{ opacity: 0, x: -20 }}
        transition={{ duration: 0.5, ease: "easeOut" }}
      >
        <Outlet />
      </motion.main>
    </div>
  );
};

export default Layout;
