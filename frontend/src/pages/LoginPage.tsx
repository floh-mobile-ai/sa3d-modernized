import React, { useState } from 'react';
import LoginForm from '../components/LoginForm';

const LoginPage: React.FC = () => {
  const [isRegisterMode, setIsRegisterMode] = useState(false);

  const toggleMode = () => {
    setIsRegisterMode(prev => !prev);
  };

  return (
    <LoginForm 
      onToggleMode={toggleMode} 
      isRegisterMode={isRegisterMode}
    />
  );
};

export default LoginPage;