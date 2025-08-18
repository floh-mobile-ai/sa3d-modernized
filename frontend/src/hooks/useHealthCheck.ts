import { useState, useEffect, useCallback } from 'react';
import type { ServiceStatus, HealthStatus } from '../types';
import { apiUtils } from '../utils/api';

export const useHealthCheck = (interval: number = 30000) => {
  const [serviceStatus, setServiceStatus] = useState<ServiceStatus>({
    apiGateway: {
      status: 'unhealthy',
      timestamp: new Date().toISOString(),
      service: 'API Gateway',
    },
    analysisService: {
      status: 'unhealthy',
      timestamp: new Date().toISOString(),
      service: 'Analysis Service',
    },
  });
  const [loading, setLoading] = useState(true);
  const [lastCheck, setLastCheck] = useState<Date | null>(null);

  const checkHealth = useCallback(async () => {
    try {
      setLoading(true);
      const timestamp = new Date().toISOString();

      // Check API Gateway health
      let apiGatewayStatus: HealthStatus = {
        status: 'unhealthy',
        timestamp,
        service: 'API Gateway',
      };

      try {
        const apiGatewayResponse = await apiUtils.getHealth();
        if (apiGatewayResponse.status === 200) {
          apiGatewayStatus = {
            status: 'healthy',
            timestamp,
            service: 'API Gateway',
            version: apiGatewayResponse.data?.version || 'unknown',
          };
        }
      } catch (error) {
        console.warn('API Gateway health check failed:', error);
      }

      // Check Analysis Service health
      let analysisServiceStatus: HealthStatus = {
        status: 'unhealthy',
        timestamp,
        service: 'Analysis Service',
      };

      try {
        const analysisResponse = await apiUtils.getAnalysisServiceHealth();
        if (analysisResponse.status === 200) {
          analysisServiceStatus = {
            status: 'healthy',
            timestamp,
            service: 'Analysis Service',
            version: analysisResponse.data?.version || 'unknown',
          };
        }
      } catch (error) {
        console.warn('Analysis Service health check failed:', error);
      }

      setServiceStatus({
        apiGateway: apiGatewayStatus,
        analysisService: analysisServiceStatus,
      });

      setLastCheck(new Date());
    } catch (error) {
      console.error('Health check error:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // Initial health check
    checkHealth();

    // Set up interval for periodic checks
    const healthCheckInterval = setInterval(checkHealth, interval);

    return () => {
      clearInterval(healthCheckInterval);
    };
  }, [checkHealth, interval]);

  return {
    serviceStatus,
    loading,
    lastCheck,
    refetch: checkHealth,
  };
};