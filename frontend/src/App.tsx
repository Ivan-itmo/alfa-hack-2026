import { useEffect, useState } from 'react';
import { clearAuth, getCurrentUser } from './api';
import { ApplicationFlowScreen } from './components/ApplicationFlowScreen';
import { DashboardScreen } from './components/DashboardScreen';
import { LoginScreen } from './components/LoginScreen';
import { Navbar } from './components/Navbar';
import { OfficerScreen } from './components/OfficerScreen';
import type { User } from './types';

export function App() {
  const [user, setUser] = useState<User | null>(getCurrentUser());
  const [activeScreen, setActiveScreen] = useState<string>('login');
  const [targetAppId, setTargetAppId] = useState<number | null>(null);

  useEffect(() => {
    const u = getCurrentUser();
    setUser(u);
    if (u) {
      setActiveScreen(u.role === 'officer' ? 'officer' : 'dashboard');
    } else {
      setActiveScreen('login');
    }

    const onAuthExpired = () => {
      setUser(null);
      setActiveScreen('login');
    };
    window.addEventListener('auth-expired', onAuthExpired);
    return () => window.removeEventListener('auth-expired', onAuthExpired);
  }, []);

  const handleLoginSuccess = (newUser: User) => {
    setUser(newUser);
    setActiveScreen(newUser.role === 'officer' ? 'officer' : 'dashboard');
  };

  const handleLogout = () => {
    clearAuth();
    setUser(null);
    setActiveScreen('login');
  };

  const handleNavigateToApplication = (appId?: number) => {
    setTargetAppId(appId || null);
    setActiveScreen('application');
  };

  return (
    <div className="app-container">
      <Navbar
        user={user}
        activeScreen={activeScreen}
        onNavigate={(screen) => setActiveScreen(screen)}
        onLogout={handleLogout}
      />

      <main className="main-content">
        {activeScreen === 'login' && (
          <LoginScreen onLoginSuccess={handleLoginSuccess} />
        )}

        {activeScreen === 'dashboard' && user && user.role === 'client' && (
          <DashboardScreen onNavigateToApplication={handleNavigateToApplication} />
        )}

        {activeScreen === 'application' && user && user.role === 'client' && (
          <ApplicationFlowScreen
            initialAppId={targetAppId}
            onNavigateToDashboard={() => setActiveScreen('dashboard')}
          />
        )}

        {activeScreen === 'officer' && user && user.role === 'officer' && (
          <OfficerScreen />
        )}
      </main>
    </div>
  );
}

export default App;
