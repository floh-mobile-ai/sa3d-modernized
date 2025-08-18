import axios, { type AxiosResponse, AxiosError } from 'axios';
import type { ApiError } from '../types';

const API_BASE_URL = 'http://localhost:8080';

// Create axios instance
export const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor to handle errors and format responses
api.interceptors.response.use(
  (response: AxiosResponse) => {
    return response;
  },
  (error: AxiosError) => {
    const apiError: ApiError = {
      message: 'An error occurred',
      status: error.response?.status || 500,
    };

    if (error.response?.data) {
      const errorData = error.response.data as any;
      apiError.message = errorData.message || errorData.error || 'Unknown error';
      apiError.code = errorData.code;
    } else if (error.message) {
      apiError.message = error.message;
    }

    // Handle 401 errors by clearing token and redirecting to login
    if (error.response?.status === 401) {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }

    return Promise.reject(apiError);
  }
);

// API utility functions
export const apiUtils = {
  // Authentication endpoints
  login: (email: string, password: string) =>
    api.post('/api/auth/login', { email, password }),
  
  register: (email: string, password: string, username: string) =>
    api.post('/api/auth/register', { email, password, username }),
  
  logout: () => api.post('/api/auth/logout'),
  
  // Health check endpoints
  getHealth: () => api.get('/health'),
  
  getAnalysisServiceHealth: () => api.get('/api/analysis/health'),
  
  // Helper to check if user is authenticated
  isAuthenticated: (): boolean => {
    const token = localStorage.getItem('auth_token');
    return !!token;
  },
  
  // Get stored user data
  getStoredUser: () => {
    const userData = localStorage.getItem('user');
    return userData ? JSON.parse(userData) : null;
  },
  
  // Store auth data
  storeAuthData: (token: string, user: any) => {
    localStorage.setItem('auth_token', token);
    localStorage.setItem('user', JSON.stringify(user));
  },
  
  // Clear auth data
  clearAuthData: () => {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('user');
  },
};