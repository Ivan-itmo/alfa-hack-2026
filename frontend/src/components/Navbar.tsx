import React from 'react';
import type { User } from '../types';

interface Props {
  user: User | null;
  activeScreen: string;
  onNavigate: (screen: string) => void;
  onLogout: () => void;
}

export const Navbar: React.FC<Props> = ({ user, activeScreen, onNavigate, onLogout }) => {
  return (
    <header className="navbar">
      <div style={{ display: 'flex', alignItems: 'center', gap: '2rem' }}>
        <a href="#" className="brand" onClick={(e) => { e.preventDefault(); onNavigate(user?.role === 'officer' ? 'officer' : 'dashboard'); }}>
          <span className="brand-badge">АЛЬФА</span>
          <span>BizCredit AI</span>
        </a>

        {user && user.role === 'client' && (
          <nav style={{ display: 'flex', gap: '0.75rem', alignItems: 'center' }}>
            <button
              className={`btn btn-sm ${activeScreen === 'dashboard' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => onNavigate('dashboard')}
            >
              Дашборд
            </button>
            <button
              className={`btn btn-sm ${activeScreen === 'applications' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => onNavigate('applications')}
            >
              Мои заявки
            </button>
            <button
              className={`btn btn-sm ${activeScreen === 'calculator' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => onNavigate('calculator')}
            >
              Калькулятор шансов
            </button>
            <button
              className={`btn btn-sm ${activeScreen === 'application' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => onNavigate('application')}
            >
              + Подать заявку
            </button>
          </nav>
        )}

        {user && user.role === 'officer' && (
          <nav style={{ display: 'flex', gap: '1rem' }}>
            <button
              className={`btn btn-sm ${activeScreen === 'officer' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => onNavigate('officer')}
            >
              Кабинет риск-офицера
            </button>
          </nav>
        )}
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
        {user ? (
          <>
            <div style={{ textAlign: 'right', fontSize: '0.85rem' }}>
              <div style={{ fontWeight: 600 }}>{user.email}</div>
              <div style={{ color: 'var(--text-muted)' }}>
                {user.role === 'officer' ? 'Риск-офицер' : `Клиент (ИНН: ${user.inn || '—'})`}
              </div>
            </div>
            <button className="btn btn-secondary btn-sm" onClick={onLogout}>
              Сменить профиль
            </button>
          </>
        ) : (
          <button className="btn btn-primary btn-sm" onClick={() => onNavigate('login')}>
            Войти
          </button>
        )}
      </div>
    </header>
  );
};
