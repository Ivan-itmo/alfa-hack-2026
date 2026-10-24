export type Role = 'client' | 'officer';

export type ApplicationStatus =
  | 'DRAFT'
  | 'DATA_ENRICHMENT'
  | 'SCORING_IN_PROGRESS'
  | 'APPROVED'
  | 'COUNTER_OFFER'
  | 'DOCS_REQUIRED'
  | 'DOCS_UPLOADED'
  | 'MANUAL_REVIEW'
  | 'REJECTED'
  | 'PENDING_SIGNATURE'
  | 'ISSUED'
  | 'CANCELLED_BY_CLIENT';

export interface User {
  id: number;
  email: string;
  role: Role;
  inn?: string;
}

export interface Company {
  id: number;
  inn: string;
  name: string;
  okved: string;
  monthly_revenue: number;
  monthly_net_profit: number;
  company_age_months: number;
  bki_active_loans_monthly_payment: number;
  bki_delinquency_days: number;
  has_bankruptcy: boolean;
  balance: number;
  revenue: number;
  registration_date?: string;
}

export interface Product {
  id: number;
  code: string;
  name: string;
  interest_rate: number;
  min_term_months: number;
  max_term_months: number;
  min_amount: number;
  max_amount: number;
}

export interface ApplicationDocument {
  id: number;
  application_id: number;
  filename: string;
  created_at: string;
}

export interface Application {
  id: number;
  user_id: number;
  company_id: number;
  company?: Company;
  documents?: ApplicationDocument[];
  product_id: number;
  purpose: string;
  status: ApplicationStatus;
  amount: number;
  term_months: number;
  requested_amount: number;
  requested_term_months: number;
  approved_amount: number;
  approved_term_months: number;
  monthly_payment: number;
  interest_rate: number;
  score_value: number;
  dsti: number;
  rejection_reason: string;
  ai_explanation: string;
  officer_comment: string;
  scoring_source: string;
  created_at: string;
  updated_at: string;
}

export interface NewsArticle {
  id: number;
  title: string;
  content: string;
  target_okved?: string;
}

export interface DashboardData {
  client: User;
  company_info: Company;
  active_application: Application | null;
  news_feed: NewsArticle[];
}
