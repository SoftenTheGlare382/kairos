import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { motion } from 'framer-motion';
import { accountApi } from '../api/client';

const Register = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const { register } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      await register(username, password);
      navigate('/login');
    } catch (err) {
      setError(err.response?.data?.error || 'Registration failed');
    }
  };

  return (
    <div className="flex h-screen items-center justify-center bg-black text-white">
      <motion.div 
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
        className="w-full max-w-md p-8 bg-surface rounded-2xl border border-border shadow-[0_0_30px_rgba(255,255,255,0.05)]"
      >
        <h2 className="text-3xl font-bold mb-8 text-center tracking-widest">KAIROS</h2>
        <p className="text-center text-textSecondary mb-8 text-sm uppercase tracking-widest">Join the Movement</p>
        
        {error && (
          <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="mb-4 p-3 bg-red-900/20 border border-red-800 text-red-400 rounded text-sm text-center"
          >
            {error}
          </motion.div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-textSecondary mb-2">Username</label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full px-4 py-3 bg-surfaceHighlight border border-border rounded-lg focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent transition-all duration-300 text-white placeholder-gray-600"
              placeholder="Choose a username"
              required
            />
          </div>
          
          <div>
            <label className="block text-sm font-medium text-textSecondary mb-2">Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-4 py-3 bg-surfaceHighlight border border-border rounded-lg focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent transition-all duration-300 text-white placeholder-gray-600"
              placeholder="Choose a password"
              required
            />
          </div>

          <button
            type="submit"
            className="w-full py-3 bg-white text-black font-bold rounded-lg hover:bg-gray-200 transition-colors duration-300 shadow-[0_0_15px_rgba(255,255,255,0.2)]"
          >
            Register
          </button>
        </form>

        <div className="mt-6 text-center text-sm text-textSecondary">
          Already have an account?{' '}
          <Link to="/login" className="text-accent hover:underline">
            Sign In
          </Link>
        </div>
      </motion.div>
    </div>
  );
};

export default Register;
