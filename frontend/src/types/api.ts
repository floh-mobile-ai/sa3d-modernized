export interface ApiResponse<T = any> {
  success: boolean;
  message: string;
  data?: T;
  error?: string;
}

export interface HealthStatus {
  status: 'healthy' | 'unhealthy';
  timestamp: string;
  service: string;
  version?: string;
}

export interface ServiceStatus {
  apiGateway: HealthStatus;
  analysisService: HealthStatus;
}

export interface ApiError {
  message: string;
  status: number;
  code?: string;
}