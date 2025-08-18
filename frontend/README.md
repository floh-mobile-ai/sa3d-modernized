# SA3D Modernized - Frontend

A modern React frontend application for the SA3D code analysis platform.

## Features

- **Authentication System**: Login/Register with JWT token management
- **Protected Dashboard**: User information and service status monitoring
- **Real-time Health Checks**: Monitor API Gateway and Analysis Service status
- **Responsive Design**: Mobile-first design with Tailwind CSS
- **Error Handling**: Comprehensive error boundaries and API error handling
- **TypeScript**: Full type safety with comprehensive interfaces

## Tech Stack

- **React 18** - Modern React with hooks and functional components
- **TypeScript** - Type safety and enhanced development experience
- **Vite** - Fast build tool and development server
- **Tailwind CSS** - Utility-first CSS framework
- **React Router** - Client-side routing and navigation
- **Axios** - HTTP client with interceptors and error handling

## Prerequisites

- Node.js 18+ and npm
- API Gateway service running on `http://localhost:8080`

## Quick Start

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

The application will be available at `http://localhost:5173/`

## Project Structure

```
frontend/
├── src/
│   ├── components/         # Reusable UI components
│   │   ├── Dashboard.tsx
│   │   ├── LoginForm.tsx
│   │   ├── ProtectedRoute.tsx
│   │   └── ErrorBoundary.tsx
│   ├── pages/             # Page components
│   │   ├── LoginPage.tsx
│   │   └── DashboardPage.tsx
│   ├── context/           # React context providers
│   │   └── AuthContext.tsx
│   ├── hooks/             # Custom React hooks
│   │   └── useHealthCheck.ts
│   ├── utils/             # Utility functions and API config
│   │   └── api.ts
│   ├── types/             # TypeScript type definitions
│   │   ├── auth.ts
│   │   ├── api.ts
│   │   └── index.ts
│   ├── App.tsx           # Main application component
│   └── main.tsx         # Application entry point
├── package.json
├── vite.config.ts
├── tailwind.config.js
└── postcss.config.js
```

## API Integration

The frontend integrates with the following backend endpoints:

- **Authentication**: 
  - `POST /api/auth/login` - User login
  - `POST /api/auth/register` - User registration
  - `POST /api/auth/logout` - User logout
- **Health Checks**:
  - `GET /health` - API Gateway health
  - `GET /api/analysis/health` - Analysis Service health

## Usage

1. **Login/Register**: Use any credentials (backend accepts any for demo)
2. **Dashboard**: View user information and service status
3. **Health Monitoring**: Real-time status of backend services
4. **Logout**: Clear session and return to login

## Development

- **Hot Reload**: Changes are reflected immediately during development
- **Type Checking**: Run `npm run build` to check for TypeScript errors
- **Error Handling**: Comprehensive error boundaries and API error handling
- **Responsive Testing**: Test on different screen sizes and devices
