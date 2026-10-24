import React, { useEffect, useState } from 'react';
import { api } from '../api';
import type { DashboardData } from '../types';

interface Props {
  onNavigateToApplication: (appId?: number) => void;
}

export const DashboardScreen: React.FC<Props> = ({ onNavigateToApplication }) => {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchDashboard = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await api.getDashboard();
      setData(res);
    } catch (err: any) {
      setError(err.message || 'Ошибка загрузки дашборда');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDashboard();
  }, []);

  const formatMoney = (amount: number) => {
    return new Intl.NumberFormat('ru-RU', {
      style: 'currency',
      currency: 'RUB',
      maximumFractionDigits: 0,
    }).format(amount);
  };

  const getStatusBadge = (status: string) => {
    const map: Record<string, { label: string; className: string }> = {
      DRAFT: { label: 'Черновик', className: 'badge-draft' },
      DATA_ENRICHMENT: { label: 'Обогащение данных', className: 'badge-scoring' },
      SCORING_IN_PROGRESS: { label: 'Скоринг в процессе', className: 'badge-scoring' },
      APPROVED: { label: 'Одобрено', className: 'badge-approved' },
      COUNTER_OFFER: { label: 'Встречное предложение', className: 'badge-counter' },
      DOCS_REQUIRED: { label: 'Требуются документы', className: 'badge-docs' },
      DOCS_UPLOADED: { label: 'Документы загружены', className: 'badge-manual' },
      MANUAL_REVIEW: { label: 'Ручное рассмотрение', className: 'badge-manual' },
      PENDING_SIGNATURE: { label: 'Ожидает подписания', className: 'badge-signature' },
      ISSUED: { label: 'Выдан (Деньги на счете)', className: 'badge-issued' },
      REJECTED: { label: 'Отказано', className: 'badge-rejected' },
      CANCELLED_BY_CLIENT: { label: 'Отменено клиентом', className: 'badge-draft' },
    };
    const s = map[status] || { label: status, className: 'badge-draft' };
    return <span className={`badge ${s.className}`}>{s.label}</span>;
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '4rem 0', color: 'var(--text-muted)' }}>
        <div style={{ fontSize: '1.2rem', fontWeight: 600 }}>Загрузка дашборда компании...</div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="card">
        <div className="alert alert-danger">{error || 'Не удалось загрузить данные'}</div>
        <button className="btn btn-primary" onClick={fetchDashboard}>Повторить попытку</button>
      </div>
    );
  }

  const { company_info, active_application, news_feed } = data;

  return (
    <div>
      {/* Top Banner: Company Profile */}
      <div className="card" style={{ background: 'linear-gradient(135deg, #1e293b 0%, #0f172a 100%)', color: 'white' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '1rem' }}>
          <div>
            <div style={{ fontSize: '0.85rem', color: '#94a3b8', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              Личный кабинет клиента
            </div>
            <h1 style={{ fontSize: '1.8rem', fontWeight: 700, margin: '0.25rem 0' }}>{company_info.name}</h1>
            <div style={{ display: 'flex', gap: '1.5rem', color: '#cbd5e1', fontSize: '0.9rem', marginTop: '0.5rem' }}>
              <span>ИНН: <strong>{company_info.inn}</strong></span>
              <span>ОКВЭД: <strong>{company_info.okved}</strong></span>
              <span>Стаж бизнеса: <strong>{company_info.company_age_months || '—'} мес.</strong></span>
            </div>
          </div>

          <div style={{ textAlign: 'right' }}>
            <div style={{ fontSize: '0.85rem', color: '#94a3b8' }}>Расчетный счет (Баланс)</div>
            <div style={{ fontSize: '2rem', fontWeight: 800, color: '#38bdf8' }}>
              {formatMoney(company_info.balance || 0)}
            </div>
          </div>
        </div>

        <div className="grid-3" style={{ marginTop: '1.5rem', borderTop: '1px solid #334155', paddingTop: '1.25rem' }}>
          <div>
            <div style={{ fontSize: '0.8rem', color: '#94a3b8' }}>Выручка в месяц</div>
            <div style={{ fontSize: '1.15rem', fontWeight: 600 }}>{formatMoney(company_info.monthly_revenue || 0)}</div>
          </div>
          <div>
            <div style={{ fontSize: '0.8rem', color: '#94a3b8' }}>Чистая прибыль в месяц</div>
            <div style={{ fontSize: '1.15rem', fontWeight: 600, color: '#4ade80' }}>
              {formatMoney(company_info.monthly_net_profit || 0)}
            </div>
          </div>
          <div>
            <div style={{ fontSize: '0.8rem', color: '#94a3b8' }}>Платежи по кредитам (БКИ)</div>
            <div style={{ fontSize: '1.15rem', fontWeight: 600, color: '#f87171' }}>
              {formatMoney(company_info.bki_active_loans_monthly_payment || 0)} / мес
            </div>
          </div>
        </div>
      </div>

      <div className="grid-2">
        {/* Left Column: Active Application */}
        <div>
          <div className="card">
            <div className="card-title">
              <span>Активная кредитная заявка</span>
              {active_application && getStatusBadge(active_application.status)}
            </div>

            {active_application ? (
              <div>
                <div className="grid-2" style={{ marginBottom: '1rem' }}>
                  <div className="stat-tile">
                    <div className="stat-label">Сумма кредита</div>
                    <div className="stat-value">{formatMoney(active_application.approved_amount || active_application.amount)}</div>
                  </div>
                  <div className="stat-tile">
                    <div className="stat-label">Срок</div>
                    <div className="stat-value">{active_application.approved_term_months || active_application.term_months} мес.</div>
                  </div>
                </div>

                <div style={{ fontSize: '0.9rem', marginBottom: '1rem', color: '#475569' }}>
                  <strong>Цель кредита:</strong> {active_application.purpose || 'Не указана'}
                </div>

                {active_application.monthly_payment > 0 && (
                  <div style={{ background: '#f0fdf4', border: '1px solid #bbf7d0', padding: '0.75rem', borderRadius: '8px', marginBottom: '1rem', fontSize: '0.9rem' }}>
                    <strong>Ежемесячный платеж:</strong> {formatMoney(active_application.monthly_payment)} (ставка: {active_application.interest_rate}%)
                  </div>
                )}

                {active_application.ai_explanation && (
                  <div style={{ background: '#eff6ff', border: '1px solid #bfdbfe', padding: '0.75rem', borderRadius: '8px', marginBottom: '1.25rem', fontSize: '0.85rem' }}>
                    <strong>Оценка скоринга:</strong> {active_application.ai_explanation}
                  </div>
                )}

                <div style={{ display: 'flex', gap: '0.75rem' }}>
                  <button
                    className="btn btn-primary"
                    style={{ flex: 1 }}
                    onClick={() => onNavigateToApplication(active_application.id)}
                  >
                    Перейти к конвейеру заявки
                  </button>
                  <button
                    className="btn btn-secondary"
                    onClick={() => onNavigateToApplication()}
                  >
                    + Новая заявка
                  </button>
                </div>
              </div>
            ) : (
              <div style={{ textAlign: 'center', padding: '2rem 1rem' }}>
                <p style={{ color: 'var(--text-muted)', marginBottom: '1.25rem' }}>
                  У вашей компании сейчас нет активных кредитных заявок.
                </p>
                <button
                  className="btn btn-primary"
                  onClick={() => onNavigateToApplication()}
                >
                  Подать заявку на кредит онлайн
                </button>
              </div>
            )}
          </div>
        </div>

        {/* Right Column: Personalized News Feed */}
        <div>
          <div className="card">
            <div className="card-title">
              <span>Персональная лента новостей</span>
              <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>Под ОКВЭД {company_info.okved}</span>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
              {news_feed && news_feed.length > 0 ? (
                news_feed.map((news) => (
                  <div
                    key={news.id}
                    style={{
                      borderLeft: news.target_okved ? '3px solid var(--primary)' : '3px solid #94a3b8',
                      paddingLeft: '1rem',
                      paddingBottom: '0.5rem',
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <h4 style={{ fontSize: '0.95rem', fontWeight: 700, color: '#1e293b' }}>{news.title}</h4>
                      {news.target_okved ? (
                        <span className="badge badge-counter" style={{ fontSize: '0.7rem' }}>ОКВЭД {news.target_okved}</span>
                      ) : (
                        <span className="badge badge-draft" style={{ fontSize: '0.7rem' }}>Общее</span>
                      )}
                    </div>
                    <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)', marginTop: '0.25rem' }}>
                      {news.content}
                    </p>
                  </div>
                ))
              ) : (
                <div style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>Новостей пока нет</div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
