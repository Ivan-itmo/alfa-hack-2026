import React, { useEffect, useState } from 'react';
import { api } from '../api';
import type { Application, ApplicationStatus, Product } from '../types';

interface Props {
  onNavigateToApplication: (appId?: number) => void;
  onNavigateToDashboard: () => void;
}

export const ApplicationsListScreen: React.FC<Props> = ({
  onNavigateToApplication,
  onNavigateToDashboard,
}) => {
  const [applications, setApplications] = useState<Application[]>([]);
  const [products, setProducts] = useState<Record<number, Product>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);
      const [appsRes, prodRes] = await Promise.all([
        api.getApplications(),
        api.getProducts().catch(() => ({ products: [] })),
      ]);

      const list = Array.isArray(appsRes) ? appsRes : appsRes.applications || [];
      setApplications(list);

      const pMap: Record<number, Product> = {};
      prodRes.products.forEach((p) => {
        pMap[p.id] = p;
      });
      setProducts(pMap);
    } catch (err: any) {
      setError(err.message || 'Ошибка загрузки реестра заявок');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const formatMoney = (amount: number) => {
    return new Intl.NumberFormat('ru-RU', {
      style: 'currency',
      currency: 'RUB',
      maximumFractionDigits: 0,
    }).format(amount);
  };

  const formatDate = (dateStr: string) => {
    try {
      const d = new Date(dateStr);
      const pad = (n: number) => n.toString().padStart(2, '0');
      return `${pad(d.getDate())}.${pad(d.getMonth() + 1)}.${d.getFullYear()}, ${pad(d.getHours())}:${pad(d.getMinutes())}`;
    } catch {
      return dateStr;
    }
  };

  const getStatusBadge = (status: ApplicationStatus) => {
    const map: Record<string, { label: string; className: string }> = {
      APPROVED: { label: 'Одобрено', className: 'bg-emerald-50 text-emerald-700 border-emerald-200' },
      ISSUED: { label: 'Выдан (Деньги на счете)', className: 'bg-emerald-50 text-emerald-700 border-emerald-200' },
      COUNTER_OFFER: { label: 'Встречное предложение', className: 'bg-amber-50 text-amber-700 border-amber-200' },
      DOCS_REQUIRED: { label: 'Требуются документы', className: 'bg-blue-50 text-blue-700 border-blue-200' },
      DOCS_UPLOADED: { label: 'Документы загружены', className: 'bg-blue-50 text-blue-700 border-blue-200' },
      MANUAL_REVIEW: { label: 'Ручное рассмотрение', className: 'bg-blue-50 text-blue-700 border-blue-200' },
      DATA_ENRICHMENT: { label: 'Обогащение данных', className: 'bg-indigo-50 text-indigo-700 border-indigo-200' },
      SCORING_IN_PROGRESS: { label: 'Скоринг в процессе', className: 'bg-indigo-50 text-indigo-700 border-indigo-200' },
      PENDING_SIGNATURE: { label: 'Ожидает подписания', className: 'bg-emerald-50 text-emerald-700 border-emerald-200' },
      REJECTED: { label: 'Отказано', className: 'bg-red-50 text-red-600 border-red-200' },
      CANCELLED_BY_CLIENT: { label: 'Отменено клиентом', className: 'bg-gray-100 text-gray-600 border-gray-200' },
      DRAFT: { label: 'Черновик', className: 'bg-gray-100 text-gray-700 border-gray-200' },
    };
    const s = map[status] || { label: status, className: 'bg-gray-100 text-gray-700 border-gray-200' };
    return (
      <span className={`inline-block px-3 py-1 text-xs font-semibold rounded-full border ${s.className}`}>
        {s.label}
      </span>
    );
  };

  const getProductName = (app: Application) => {
    if (products[app.product_id]) {
      return products[app.product_id].name;
    }
    switch (app.product_id) {
      case 1:
        return 'Овердрафт';
      case 2:
        return 'Оборотный кредит';
      case 3:
        return 'Инвестиционный кредит';
      default:
        return 'Кредит для бизнеса';
    }
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '4rem 0', color: 'var(--text-muted)' }}>
        <div style={{ fontSize: '1.2rem', fontWeight: 600 }}>Загрузка реестра ваших заявок...</div>
      </div>
    );
  }

  return (
    <div style={{ maxWidth: '1060px', margin: '0 auto', display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
      {/* Top Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '1rem' }}>
        <div>
          <h1 style={{ fontSize: '1.8rem', fontWeight: 800, margin: '0 0 0.25rem 0' }}>
            Мои кредитные заявки
          </h1>
          <p style={{ color: 'var(--text-muted)', margin: 0, fontSize: '0.95rem' }}>
            История рассмотрения, действующие решения и статусы договоров
          </p>
        </div>
        <div style={{ display: 'flex', gap: '0.75rem', alignItems: 'center' }}>
          <button className="btn btn-secondary" onClick={onNavigateToDashboard}>
            ← В дашборд
          </button>
          <button
            className="btn btn-primary"
            style={{ fontWeight: 700 }}
            onClick={() => onNavigateToApplication()}
          >
            + Оформить новую заявку
          </button>
        </div>
      </div>

      {error && (
        <div className="alert alert-danger" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span>{error}</span>
          <button className="btn btn-sm btn-secondary" onClick={loadData}>Повторить</button>
        </div>
      )}

      {/* Applications List */}
      {applications.length === 0 ? (
        <div className="card" style={{ textAlign: 'center', padding: '3.5rem 1.5rem' }}>
          <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>📑</div>
          <h2 style={{ fontSize: '1.4rem', fontWeight: 700, color: '#0f172a', marginBottom: '0.5rem' }}>
            У вас пока нет активных заявок
          </h2>
          <p style={{ color: 'var(--text-muted)', maxWidth: '480px', margin: '0 auto 1.5rem', lineHeight: 1.5 }}>
            Оформите первую заявку на экспресс-финансирование онлайн — искусственный интеллект оценит бизнес за 2 минуты без сбора справок.
          </p>
          <button
            className="btn btn-primary"
            style={{ padding: '0.75rem 1.75rem', fontSize: '1rem', fontWeight: 600 }}
            onClick={() => onNavigateToApplication()}
          >
            + Подать первую заявку
          </button>
        </div>
      ) : (
        <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
          <div style={{ padding: '1.25rem 1.5rem', borderBottom: '1px solid var(--border)', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '0.5rem' }}>
            <div style={{ fontWeight: 700, fontSize: '1.1rem', color: '#0f172a' }}>
              Всего заявок в системе: {applications.length}
            </div>
            <span className="badge badge-secondary" style={{ fontSize: '0.8rem' }}>
              Обновлено в реальном времени
            </span>
          </div>

          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '0.9rem' }}>
              <thead>
                <tr style={{ background: '#f8fafc', borderBottom: '1px solid var(--border)', color: '#64748b', fontSize: '0.78rem', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                  <th style={{ padding: '0.85rem 1.25rem' }}>Заявка / Дата</th>
                  <th style={{ padding: '0.85rem 1.25rem' }}>Продукт</th>
                  <th style={{ padding: '0.85rem 1.25rem' }}>Запрошено</th>
                  <th style={{ padding: '0.85rem 1.25rem' }}>Решение / Платеж</th>
                  <th style={{ padding: '0.85rem 1.25rem' }}>Статус</th>
                  <th style={{ padding: '0.85rem 1.25rem', textAlign: 'right' }}>Действие</th>
                </tr>
              </thead>
              <tbody>
                {applications.map((app) => {
                  const hasOffer = app.approved_amount > 0 || (app.status === 'COUNTER_OFFER' && app.approved_amount > 0);
                  const displayOffer = app.approved_amount;
                  return (
                    <tr
                      key={app.id}
                      style={{ borderBottom: '1px solid #f1f5f9', transition: 'background-color 0.15s' }}
                      className="hover:bg-gray-50"
                    >
                      {/* ID and Date */}
                      <td style={{ padding: '1rem 1.25rem' }}>
                        <div style={{ fontWeight: 800, color: '#0f172a', fontSize: '0.95rem' }}>
                          #{app.id}
                        </div>
                        <div style={{ fontSize: '0.78rem', color: '#94a3b8', marginTop: '0.2rem' }}>
                          {formatDate(app.created_at)}
                        </div>
                      </td>

                      {/* Product */}
                      <td style={{ padding: '1rem 1.25rem' }}>
                        <div style={{ fontWeight: 600, color: '#1e293b' }}>
                          {getProductName(app)}
                        </div>
                        <div style={{ fontSize: '0.78rem', color: '#64748b', marginTop: '0.15rem' }}>
                          {app.purpose ? (app.purpose.length > 32 ? app.purpose.slice(0, 32) + '...' : app.purpose) : 'Финансирование бизнеса'}
                        </div>
                      </td>

                      {/* Requested */}
                      <td style={{ padding: '1rem 1.25rem' }}>
                        <div style={{ fontWeight: 700, color: '#0f172a' }}>
                          {formatMoney(app.requested_amount || app.amount)}
                        </div>
                        <div style={{ fontSize: '0.78rem', color: '#64748b', marginTop: '0.15rem' }}>
                          на {app.requested_term_months || app.term_months} мес.
                        </div>
                      </td>

                      {/* Approved / Offer */}
                      <td style={{ padding: '1rem 1.25rem' }}>
                        {hasOffer ? (
                          <div>
                            <div style={{ fontWeight: 800, color: '#059669', fontSize: '0.95rem' }}>
                              {formatMoney(displayOffer)}
                            </div>
                            <div style={{ fontSize: '0.78rem', color: '#047857', marginTop: '0.15rem' }}>
                              {app.monthly_payment > 0 ? `${formatMoney(app.monthly_payment)}/мес` : `${app.approved_term_months || app.term_months} мес.`}
                            </div>
                          </div>
                        ) : (
                          <div style={{ color: '#94a3b8', fontSize: '0.85rem' }}>
                            {app.status === 'REJECTED' ? '—' : 'На оценке'}
                          </div>
                        )}
                      </td>

                      {/* Status */}
                      <td style={{ padding: '1rem 1.25rem' }}>
                        {getStatusBadge(app.status)}
                      </td>

                      {/* Action */}
                      <td style={{ padding: '1rem 1.25rem', textAlign: 'right' }}>
                        <button
                          className="btn btn-sm btn-primary"
                          style={{ fontWeight: 600, padding: '0.35rem 0.85rem' }}
                          onClick={() => onNavigateToApplication(app.id)}
                        >
                          Открыть →
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};
