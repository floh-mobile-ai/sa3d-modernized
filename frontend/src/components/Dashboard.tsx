import React from 'react';
import { useAuth } from '../context/AuthContext';
import { useHealthCheck } from '../hooks/useHealthCheck';
import type { HealthStatus } from '../types';

const StatusIndicator: React.FC<{ status: HealthStatus }> = ({ status }) => {
  const isHealthy = status.status === 'healthy';
  
  return (
    <div className="bg-white overflow-hidden shadow rounded-lg">
      <div className="p-5">
        <div className="flex items-center">
          <div className="flex-shrink-0">
            <div className={`h-4 w-4 rounded-full ${isHealthy ? 'bg-green-400' : 'bg-red-400'}`}>
              {isHealthy && (
                <div className="h-4 w-4 rounded-full bg-green-400 animate-pulse"></div>
              )}
            </div>
          </div>
          <div className="ml-5 w-0 flex-1">
            <dl>
              <dt className="text-sm font-medium text-gray-500 truncate">
                {status.service}
              </dt>
              <dd className="text-lg font-medium text-gray-900">
                {isHealthy ? 'Healthy' : 'Unhealthy'}
              </dd>
            </dl>
          </div>
        </div>
        <div className="mt-4">
          <div className="text-sm text-gray-600">
            <p>Last checked: {new Date(status.timestamp).toLocaleTimeString()}</p>
            {status.version && (
              <p className="mt-1">Version: {status.version}</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

const Dashboard: React.FC = () => {
  const { user, logout } = useAuth();
  const { serviceStatus, loading, lastCheck, refetch } = useHealthCheck();

  const handleLogout = () => {
    logout();
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Navigation */}
      <nav className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <h1 className="text-xl font-semibold text-gray-900">
                SA3D Modernized
              </h1>
            </div>
            <div className="flex items-center space-x-4">
              <span className="text-sm text-gray-700">
                Welcome, {user?.username || user?.email}
              </span>
              <button
                onClick={handleLogout}
                className="bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-md text-sm font-medium transition-colors duration-200"
              >
                Logout
              </button>
            </div>
          </div>
        </div>
      </nav>

      {/* Main content */}
      <div className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        {/* Welcome section */}
        <div className="px-4 py-6 sm:px-0">
          <div className="bg-white overflow-hidden shadow rounded-lg">
            <div className="px-4 py-5 sm:p-6">
              <h2 className="text-2xl font-bold text-gray-900 mb-4">
                Welcome to SA3D Modernized
              </h2>
              <p className="text-gray-600 mb-4">
                Code Analysis Platform - Dashboard
              </p>
              <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
                <div className="flex">
                  <div className="flex-shrink-0">
                    <svg className="h-5 w-5 text-blue-400" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                    </svg>
                  </div>
                  <div className="ml-3">
                    <h3 className="text-sm font-medium text-blue-800">
                      User Information
                    </h3>
                    <div className="mt-2 text-sm text-blue-700">
                      <p><strong>Email:</strong> {user?.email}</p>
                      <p><strong>Username:</strong> {user?.username}</p>
                      <p><strong>User ID:</strong> {user?.id}</p>
                      {user?.createdAt && (
                        <p><strong>Member since:</strong> {new Date(user.createdAt).toLocaleDateString()}</p>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Service Status section */}
        <div className="px-4 py-6 sm:px-0">
          <div className="flex justify-between items-center mb-6">
            <h3 className="text-lg leading-6 font-medium text-gray-900">
              Service Status
            </h3>
            <div className="flex items-center space-x-4">
              {lastCheck && (
                <span className="text-sm text-gray-500">
                  Last updated: {lastCheck.toLocaleTimeString()}
                </span>
              )}
              <button
                onClick={refetch}
                disabled={loading}
                className="bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-400 text-white px-4 py-2 rounded-md text-sm font-medium transition-colors duration-200"
              >
                {loading ? (
                  <span className="flex items-center">
                    <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    Checking...
                  </span>
                ) : (
                  'Refresh Status'
                )}
              </button>
            </div>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <StatusIndicator status={serviceStatus.apiGateway} />
            <StatusIndicator status={serviceStatus.analysisService} />
          </div>
        </div>

        {/* Additional Information */}
        <div className="px-4 py-6 sm:px-0">
          <div className="bg-white overflow-hidden shadow rounded-lg">
            <div className="px-4 py-5 sm:p-6">
              <h3 className="text-lg font-medium text-gray-900 mb-4">
                Platform Information
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <h4 className="font-medium text-gray-700">API Gateway</h4>
                  <p className="text-sm text-gray-600">http://localhost:8080</p>
                  <p className="text-sm text-gray-500">Handles authentication and request routing</p>
                </div>
                <div>
                  <h4 className="font-medium text-gray-700">Analysis Service</h4>
                  <p className="text-sm text-gray-600">Available through API Gateway</p>
                  <p className="text-sm text-gray-500">Provides code analysis capabilities</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;