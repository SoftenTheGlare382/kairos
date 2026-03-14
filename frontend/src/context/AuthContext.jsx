import React, { createContext, useContext, useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api, { accountApi } from '../api/client';

const AuthContext = createContext();

export const useAuth = () => useContext(AuthContext);

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('token'));
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    const initAuth = async () => {
      if (token) {
        localStorage.setItem('token', token);
        // Try to recover user info from token or local storage
        // Since we don't have a "me" endpoint that returns ID from token directly (except internal validate),
        // we might need to rely on stored user info or just keep the token.
        // For better UX, let's try to decode token if possible, or just wait for a 401 to logout.
        // But we need user.id for many things.
        
        // Strategy: Store user info in localStorage as well, or fetch by username if we stored username.
        const storedUser = localStorage.getItem('user');
        if (storedUser) {
          setUser(JSON.parse(storedUser));
        }
      } else {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        setUser(null);
      }
      setLoading(false);
    };
    initAuth();
  }, [token]);

  const login = async (username, password) => {
    try {
      const response = await accountApi.login(username, password);
      const { token: newToken } = response.data;
      
      setToken(newToken);
      localStorage.setItem('token', newToken);

      // Fetch user details to get ID
      // We need to set the token in header first (interceptor handles this via localStorage or variable)
      // But interceptor reads from localStorage.getItem('token').
      
      try {
        const userRes = await accountApi.findByUsername(username);
        const userData = userRes.data;
        setUser(userData);
        localStorage.setItem('user', JSON.stringify(userData));
      } catch (e) {
        console.error("Failed to fetch user details", e);
        // Fallback: create a temporary user object if fetch fails (shouldn't happen if login ok)
        // But findByUsername requires auth.
      }

      navigate('/');
      return true;
    } catch (error) {
      console.error("Login failed", error);
      throw error;
    }
  };

  const register = async (username, password) => {
    try {
      await accountApi.register(username, password);
      return true;
    } catch (error) {
      console.error("Registration failed", error);
      throw error;
    }
  };

  const logout = () => {
    setToken(null);
    setUser(null);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    navigate('/login');
  };

  const value = {
    user,
    token,
    login,
    register,
    logout,
    loading
  };

  return (
    <AuthContext.Provider value={value}>
      {!loading && children}
    </AuthContext.Provider>
  );
};
