import React, { createContext, useContext, useReducer, useEffect } from 'react';
import type { AuthContextType, User, ApiError } from '../types';
import { apiUtils } from '../utils/api';

interface AuthState {
  user: User | null;
  token: string | null;
  loading: boolean;
  error: string | null;
}

type AuthAction =
  | { type: 'LOGIN_START' }
  | { type: 'LOGIN_SUCCESS'; payload: { user: User; token: string } }
  | { type: 'LOGIN_ERROR'; payload: string }
  | { type: 'LOGOUT' }
  | { type: 'CLEAR_ERROR' }
  | { type: 'SET_LOADING'; payload: boolean };

const initialState: AuthState = {
  user: null,
  token: null,
  loading: true,
  error: null,
};

const authReducer = (state: AuthState, action: AuthAction): AuthState => {
  switch (action.type) {
    case 'LOGIN_START':
      return { ...state, loading: true, error: null };
    case 'LOGIN_SUCCESS':
      return {
        ...state,
        user: action.payload.user,
        token: action.payload.token,
        loading: false,
        error: null,
      };
    case 'LOGIN_ERROR':
      return { ...state, loading: false, error: action.payload };
    case 'LOGOUT':
      return { ...initialState, loading: false };
    case 'CLEAR_ERROR':
      return { ...state, error: null };
    case 'SET_LOADING':
      return { ...state, loading: action.payload };
    default:
      return state;
  }
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [state, dispatch] = useReducer(authReducer, initialState);

  // Check for existing auth on mount
  useEffect(() => {
    const token = localStorage.getItem('auth_token');
    const user = apiUtils.getStoredUser();
    
    if (token && user) {
      dispatch({ type: 'LOGIN_SUCCESS', payload: { token, user } });
    } else {
      dispatch({ type: 'SET_LOADING', payload: false });
    }
  }, []);

  const login = async (email: string, password: string): Promise<void> => {
    dispatch({ type: 'LOGIN_START' });
    
    try {
      const response = await apiUtils.login(email, password);
      const { token, user } = response.data;
      
      // Store auth data
      apiUtils.storeAuthData(token, user);
      
      dispatch({ type: 'LOGIN_SUCCESS', payload: { token, user } });
    } catch (error) {
      const apiError = error as ApiError;
      dispatch({ type: 'LOGIN_ERROR', payload: apiError.message });
      throw error;
    }
  };

  const register = async (email: string, password: string, username: string): Promise<void> => {
    dispatch({ type: 'LOGIN_START' });
    
    try {
      const response = await apiUtils.register(email, password, username);
      const { token, user } = response.data;
      
      // Store auth data
      apiUtils.storeAuthData(token, user);
      
      dispatch({ type: 'LOGIN_SUCCESS', payload: { token, user } });
    } catch (error) {
      const apiError = error as ApiError;
      dispatch({ type: 'LOGIN_ERROR', payload: apiError.message });
      throw error;
    }
  };

  const logout = (): void => {
    // Clear stored data
    apiUtils.clearAuthData();
    
    // Call logout endpoint (fire and forget)
    apiUtils.logout().catch(() => {
      // Ignore errors for logout endpoint
    });
    
    dispatch({ type: 'LOGOUT' });
  };

  const value: AuthContextType = {
    user: state.user,
    token: state.token,
    login,
    register,
    logout,
    loading: state.loading,
    error: state.error,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};