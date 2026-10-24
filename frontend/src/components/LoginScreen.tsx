import React, { useState } from 'react';
import { api, setAuth } from '../api';
import type { User } from '../types';

interface Props {
  onLoginSuccess: (user: User) => void;
}

export const LoginScreen: React.FC<Props> = ({ onLoginSuccess }) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const demoUsers = [
    {
      id: 'client1',
      title: 'client1 (Альфа-Снаб)',
      company: 'ООО "Стабильный Бизнес"',
      inn: '7701000001',
      okved: '62.01 (IT / Разработка)',
      revenue: '24 000 000 ₽ / год',
      profit: '600 000 ₽ / мес',
      badge: 'Клиент',
      badgeClass: 'badge-approved',
      hint: 'Зрелый бизнес (8 лет). Скоринг проходит моментально с автоматическим одобрением.',
    },
    {
      id: 'client2',
      title: 'client2 (ИП Смирнов)',
      company: 'ИП "Ритейл"',
      inn: '770100000002',
      okved: '47.11 (Торговля в магазинах)',
      revenue: '12 000 000 ₽ / год',
      profit: '180 000 ₽ / мес',
      badge: 'Клиент',
      badgeClass: 'badge-counter',
      hint: 'Торговый бизнес (5 лет). При крупной заявке срабатывает встречное предложение (DSTI > 50%).',
    },
    {
      id: 'client3',
      title: 'client3 (Стартап)',
      company: 'ИП "Новый"',
      inn: '770100000003',
      okved: '62.01 (IT / Стартап)',
      revenue: '600 000 ₽ / год',
      profit: '10 000 ₽ / мес',
      badge: 'Клиент',
      badgeClass: 'badge-docs',
      hint: 'Молодой бизнес (1 месяц). Требует дозапроса фин. отчетности и ручной проверки.',
    },
    {
      id: 'officer1',
      title: 'officer1 (Кредитный офицер)',
      company: 'Альфа-Банк (Центральный офис)',
      inn: 'Внутренний аккаунт',
      okved: 'Риск-менеджмент и андеррайтинг',
      revenue: '—',
      profit: '—',
      badge: 'Сотрудник',
      badgeClass: 'badge-manual',
      hint: 'Доступ ко всем заявкам, ручной анализ, вынесение решений и встречных предложений.',
    },
  ];

  const handleLogin = async (email: string) => {
    try {
      setLoading(true);
      setError(null);
      const res = await api.login(email, '123');
      setAuth(res.token, res.user);
      onLoginSuccess(res.user);
    } catch (err: any) {
      setError(err.message || 'Ошибка входа');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ maxWidth: '900px', margin: '2rem auto' }}>
      <div style={{ textAlign: 'center', marginBottom: '2.5rem' }}>
        <h1 style={{ fontSize: '2.2rem', fontWeight: 800, marginBottom: '0.75rem', color: '#0f172a' }}>
          Умный кредитный кабинет предпринимателя
        </h1>
        <p style={{ color: 'var(--text-muted)', fontSize: '1.1rem' }}>
          Интеллектуальный скоринг кредитных заявок бизнеса, AI-советник и онлайн-выдача Альфа-Банка.
        </p>
      </div>

      {error && <div className="alert alert-danger">{error}</div>}

      <div className="card" style={{ marginBottom: '2rem' }}>
        <div className="card-title">
          <span>Выберите демо-профиль для мгновенного входа</span>
          <span style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>Пароль по умолчанию: 123</span>
        </div>

        <div className="grid-2">
          {demoUsers.map((u) => (
            <div
              key={u.id}
              className="demo-card"
              onClick={() => !loading && handleLogin(u.id)}
            >
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span className="demo-role">{u.badge}</span>
                  <span className={`badge ${u.badgeClass}`}>{u.id}</span>
                </div>
                <h3 className="demo-name">{u.title}</h3>
                <div style={{ fontSize: '0.88rem', fontWeight: 600, color: '#334155' }}>{u.company}</div>
                <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '0.2rem' }}>
                  ИНН: {u.inn} • ОКВЭД: {u.okved}
                </div>
                <p className="demo-desc" style={{ marginTop: '0.6rem' }}>{u.hint}</p>
              </div>

              <button
                className="btn btn-primary btn-sm"
                style={{ width: '100%', marginTop: '0.5rem' }}
                disabled={loading}
              >
                {loading ? 'Вход...' : `Войти как ${u.id}`}
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
