import React, { useEffect, useMemo, useState } from 'react';
import { api } from '../api';
import type { DashboardData } from '../types';

interface Props {
  onNavigateToApplication: (appId?: number, params?: { amount?: number; termMonths?: number }) => void;
  onNavigateToDashboard: () => void;
}

const AMOUNT_CHIPS = [
  { label: '300 тыс ₽', value: 300000 },
  { label: '1 млн ₽', value: 1000000 },
  { label: '2 млн ₽', value: 2000000 },
  { label: '3 млн ₽', value: 3000000 },
  { label: '5 млн ₽', value: 5000000 },
];

const TERM_CHIPS = [
  { label: '6 мес', value: 6 },
  { label: '12 мес', value: 12 },
  { label: '24 мес', value: 24 },
  { label: '36 мес', value: 36 },
];

export const CalculatorScreen: React.FC<Props> = ({ onNavigateToApplication, onNavigateToDashboard }) => {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Calibrated Credit Potential Calculator State
  const [calcAmount, setCalcAmount] = useState<number>(1500000);
  const [calcTerm, setCalcTerm] = useState<number>(12);

  const fetchCompanyData = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await api.getDashboard();
      setData(res);
    } catch (err: any) {
      setError(err.message || 'Ошибка загрузки финансовых данных компании');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCompanyData();
  }, []);

  const formatMoney = (amount: number) => {
    return new Intl.NumberFormat('ru-RU', {
      style: 'currency',
      currency: 'RUB',
      maximumFractionDigits: 0,
    }).format(amount);
  };

  // Real-time calculation logic
  const { pmt, dsti, trafficLight } = useMemo(() => {
    const rate = 0.20; // 20% annual
    const r = rate / 12;
    const n = calcTerm;
    const s = calcAmount;

    // PMT = S * (r * (1 + r)^n) / ((1 + r)^n - 1)
    const factor = Math.pow(1 + r, n);
    const calculatedPmt = Math.round((s * (r * factor)) / (factor - 1));

    const profit =
      data?.company_info.monthly_net_profit && data.company_info.monthly_net_profit > 0
        ? data.company_info.monthly_net_profit
        : data?.company_info.monthly_revenue
        ? data.company_info.monthly_revenue * 0.25
        : 600000;

    const bki = data?.company_info.bki_active_loans_monthly_payment || 0;
    const calculatedDsti = (bki + calculatedPmt) / profit;

    let light: {
      type: 'green' | 'yellow' | 'red';
      badge: string;
      bgClass: string;
      borderClass: string;
      textClass: string;
      hint: string;
    };

    const dstiPct = (calculatedDsti * 100).toFixed(1);

    if (calculatedDsti <= 0.35) {
      light = {
        type: 'green',
        badge: '🟢 Высокий шанс экспресс-одобрения',
        bgClass: 'bg-emerald-50',
        borderClass: 'border-emerald-200',
        textClass: 'text-emerald-900',
        hint: `Платеж комфортен для вашего бизнеса (DSTI ${dstiPct}%). Скоринг займет менее 2 минут.`,
      };
    } else if (calculatedDsti <= 0.55) {
      light = {
        type: 'yellow',
        badge: '🟡 Встречное предложение (Counter-Offer)',
        bgClass: 'bg-amber-50',
        borderClass: 'border-amber-200',
        textClass: 'text-amber-900',
        hint: `Платеж на границе норматива (DSTI ${dstiPct}%). Банк может предложить сниженную сумму или увеличить срок.`,
      };
    } else {
      light = {
        type: 'red',
        badge: '🔴 Высокая долговая нагрузка',
        bgClass: 'bg-red-50',
        borderClass: 'border-red-200',
        textClass: 'text-red-900',
        hint: `Рекомендуем увеличить срок кредита или запросить меньшую сумму во избежание кассового разрыва (DSTI ${dstiPct}%).`,
      };
    }

    return {
      pmt: calculatedPmt,
      dsti: calculatedDsti,
      trafficLight: light,
    };
  }, [calcAmount, calcTerm, data?.company_info]);

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '4rem 0', color: 'var(--text-muted)' }}>
        <div style={{ fontSize: '1.2rem', fontWeight: 600 }}>Загрузка финансового профиля компании...</div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="card" style={{ maxWidth: '800px', margin: '0 auto' }}>
        <div className="alert alert-danger">{error || 'Не удалось загрузить данные'}</div>
        <button className="btn btn-primary" onClick={fetchCompanyData}>Повторить попытку</button>
      </div>
    );
  }

  const { company_info } = data;

  return (
    <div style={{ maxWidth: '960px', margin: '0 auto', display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
      {/* Top Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '1rem' }}>
        <div>
          <h1 style={{ fontSize: '1.8rem', fontWeight: 800, margin: '0 0 0.25rem 0' }}>
            Калькулятор кредитного потенциала
          </h1>
          <p style={{ color: 'var(--text-muted)', margin: 0, fontSize: '0.95rem' }}>
            Оценка вероятности одобрения (DSTI) и расчет ежемесячного платежа для МСП
          </p>
        </div>
        <button className="btn btn-secondary" onClick={onNavigateToDashboard}>
          ← В дашборд
        </button>
      </div>

      {/* Financial Health Summary Tile */}
      <div className="card" style={{ background: 'linear-gradient(135deg, #1e293b 0%, #0f172a 100%)', color: 'white', marginBottom: 0 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '1rem', borderBottom: '1px solid #334155', paddingBottom: '1rem' }}>
          <div>
            <div style={{ fontSize: '0.8rem', color: '#94a3b8', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              Финансовый профиль компании
            </div>
            <div style={{ fontSize: '1.3rem', fontWeight: 700, marginTop: '0.2rem' }}>
              {company_info.name}
            </div>
            <div style={{ fontSize: '0.85rem', color: '#cbd5e1', marginTop: '0.2rem' }}>
              ИНН: <strong>{company_info.inn}</strong> • ОКВЭД: <strong>{company_info.okved}</strong>
            </div>
          </div>
          <div style={{ textAlign: 'right' }}>
            <div style={{ fontSize: '0.8rem', color: '#94a3b8' }}>Остаток на счете</div>
            <div style={{ fontSize: '1.5rem', fontWeight: 800, color: '#38bdf8' }}>
              {formatMoney(company_info.balance || 0)}
            </div>
          </div>
        </div>

        <div className="grid-3" style={{ paddingTop: '1rem' }}>
          <div>
            <div style={{ fontSize: '0.8rem', color: '#94a3b8' }}>Выручка в месяц</div>
            <div style={{ fontSize: '1.1rem', fontWeight: 600 }}>{formatMoney(company_info.monthly_revenue || 0)}</div>
          </div>
          <div>
            <div style={{ fontSize: '0.8rem', color: '#94a3b8' }}>Чистая прибыль (База DSTI)</div>
            <div style={{ fontSize: '1.1rem', fontWeight: 600, color: '#4ade80' }}>
              {formatMoney(company_info.monthly_net_profit || 0)}
            </div>
          </div>
          <div>
            <div style={{ fontSize: '0.8rem', color: '#94a3b8' }}>Платежи по кредитам (БКИ)</div>
            <div style={{ fontSize: '1.1rem', fontWeight: 600, color: '#f87171' }}>
              {formatMoney(company_info.bki_active_loans_monthly_payment || 0)} / мес
            </div>
          </div>
        </div>
      </div>

      {/* Main Interactive Calculator Card */}
      <div className="card calculator-card" style={{ marginBottom: 0 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '0.75rem', marginBottom: '1.5rem' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem' }}>
            <span style={{ fontSize: '1.4rem' }}>🧮</span>
            <div>
              <h2 style={{ fontSize: '1.25rem', fontWeight: 800, color: '#0f172a', margin: 0 }}>
                Параметры беззалогового экспресс-кредитования
              </h2>
              <div style={{ fontSize: '0.82rem', color: 'var(--text-muted)', marginTop: '0.2rem' }}>
                До 5 млн ₽ без визита в отделение • Решение онлайн за 2 минуты
              </div>
            </div>
          </div>
          <span className="badge badge-primary" style={{ padding: '0.35rem 0.75rem', fontSize: '0.82rem' }}>
            Базовая ставка 20% годовых
          </span>
        </div>

        <div className="calculator-grid">
          {/* Controls Column */}
          <div className="calculator-controls">
            {/* Amount Slider */}
            <div className="form-group" style={{ marginBottom: '1.5rem' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
                <label className="form-label" style={{ margin: 0, fontWeight: 700, fontSize: '0.95rem' }}>
                  Сумма финансирования:
                </label>
                <span style={{ fontSize: '1.35rem', fontWeight: 800, color: 'var(--primary)' }}>
                  {formatMoney(calcAmount)}
                </span>
              </div>
              <input
                type="range"
                className="form-control"
                style={{ padding: '0.2rem', cursor: 'pointer' }}
                min={100000}
                max={5000000}
                step={50000}
                value={calcAmount}
                onChange={(e) => setCalcAmount(Number(e.target.value))}
              />
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '0.3rem' }}>
                <span>100 тыс ₽</span>
                <span>Лимит до 5 млн ₽</span>
              </div>
              <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.6rem', flexWrap: 'wrap' }}>
                {AMOUNT_CHIPS.map((chip) => (
                  <button
                    key={chip.value}
                    type="button"
                    className={`btn btn-sm ${calcAmount === chip.value ? 'btn-primary' : 'btn-secondary'}`}
                    style={{ fontSize: '0.8rem', padding: '0.3rem 0.75rem', fontWeight: 600 }}
                    onClick={() => setCalcAmount(chip.value)}
                  >
                    {chip.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Term Slider */}
            <div className="form-group" style={{ marginBottom: '0.5rem' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
                <label className="form-label" style={{ margin: 0, fontWeight: 700, fontSize: '0.95rem' }}>
                  Срок кредита:
                </label>
                <span style={{ fontSize: '1.35rem', fontWeight: 800, color: 'var(--primary)' }}>
                  {calcTerm} мес.
                </span>
              </div>
              <input
                type="range"
                className="form-control"
                style={{ padding: '0.2rem', cursor: 'pointer' }}
                min={6}
                max={36}
                step={6}
                value={calcTerm}
                onChange={(e) => setCalcTerm(Number(e.target.value))}
              />
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '0.3rem' }}>
                <span>6 мес.</span>
                <span>до 36 мес. (шаг 6 мес.)</span>
              </div>
              <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.6rem', flexWrap: 'wrap' }}>
                {TERM_CHIPS.map((chip) => (
                  <button
                    key={chip.value}
                    type="button"
                    className={`btn btn-sm ${calcTerm === chip.value ? 'btn-primary' : 'btn-secondary'}`}
                    style={{ fontSize: '0.8rem', padding: '0.3rem 0.75rem', fontWeight: 600 }}
                    onClick={() => setCalcTerm(chip.value)}
                  >
                    {chip.label}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Results & Traffic Light Column */}
          <div className="calculator-results" style={{ display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
            <div className="calc-summary-tile">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: '0.5rem' }}>
                <span style={{ fontSize: '0.9rem', color: 'var(--text-muted)' }}>Прогнозный ежемесячный платеж:</span>
                <span style={{ fontSize: '1.65rem', fontWeight: 800, color: '#0f172a' }}>
                  {formatMoney(pmt)} <span style={{ fontSize: '0.9rem', fontWeight: 500, color: '#64748b' }}>/ мес</span>
                </span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.8rem', color: 'var(--text-muted)', borderTop: '1px solid #f1f5f9', paddingTop: '0.5rem', flexWrap: 'wrap', gap: '0.5rem' }}>
                <span>Платежи БКИ: <strong>{formatMoney(company_info.bki_active_loans_monthly_payment || 0)}</strong></span>
                <span>Чистая прибыль: <strong>{formatMoney(company_info.monthly_net_profit || 0)}</strong></span>
              </div>
            </div>

            {/* Traffic Light Card */}
            <div className={`calc-traffic-light ${trafficLight.bgClass} ${trafficLight.borderClass} ${trafficLight.textClass}`} style={{ marginBottom: '1rem' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem' }}>
                <span style={{ fontWeight: 700, fontSize: '0.95rem' }}>
                  {trafficLight.badge}
                </span>
                <span style={{ fontWeight: 800, fontSize: '1rem' }}>
                  DSTI: {(dsti * 100).toFixed(1)}%
                </span>
              </div>
              <p style={{ margin: 0, fontSize: '0.85rem', lineHeight: 1.5 }}>
                {trafficLight.hint}
              </p>
            </div>

            {/* Action CTA Button */}
            <button
              type="button"
              className="btn btn-primary"
              style={{ width: '100%', padding: '0.85rem', fontWeight: 700, fontSize: '1.05rem' }}
              onClick={() => onNavigateToApplication(undefined, { amount: calcAmount, termMonths: calcTerm })}
            >
              Оформить с этими параметрами →
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
