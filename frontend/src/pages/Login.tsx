import { useState, type FormEvent } from 'react';
import { ArrowRight, Check, Layers3, Sparkles } from 'lucide-react';
import { api } from '../api/client';
import type { Session } from '../api/types';
import { Brand, ErrorMessage } from '../components/Shared';
export default function Login({ onLogin }: { onLogin: (session: Session) => void }) {
  const [role, setRole] = useState<'employee' | 'hr'>('employee');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError('');
    try {
      onLogin(await api.login(role, password));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="login-page">
      <div className="login-story">
        <Brand />
        <div className="story-content">
          <span className="eyebrow light">
            <Sparkles size={15} /> ВАША КАРЬЕРА. ВАША ТРАЕКТОРИЯ.
          </span>
          <h1>
            Большой рост
            <br />
            начинается
            <br />
            со <em>следующего шага.</em>
          </h1>
          <p>
            Навыки, возможности и понятный путь к вашей цели. Развивайтесь осознанно — в своём
            темпе.
          </p>
          <div className="story-path">
            <span>
              <Check size={18} />
            </span>
            <i />
            <span>
              <Layers3 size={19} />
            </span>
            <i />
            <span className="path-future">
              <Sparkles size={18} />
            </span>
          </div>
          <div className="path-labels">
            <span>Ваш опыт</span>
            <span>Следующий шаг</span>
            <span>Новая роль</span>
          </div>
        </div>
        <div className="story-footer">
          CAREER QUEST <span>HACKALEM AI · 2026</span>
        </div>
      </div>
      <div className="login-form-wrap">
        <div className="login-form">
          <span className="eyebrow">РАДЫ ВАС ВИДЕТЬ</span>
          <h2>Продолжим ваш путь</h2>
          <p className="muted">Войдите в пространство развития.</p>
          <div className="role-toggle">
            <button
              type="button"
              className={role === 'employee' ? 'selected' : ''}
              aria-pressed={role === 'employee'}
              onClick={() => setRole('employee')}
            >
              Сотрудник
            </button>
            <button
              type="button"
              className={role === 'hr' ? 'selected' : ''}
              aria-pressed={role === 'hr'}
              onClick={() => setRole('hr')}
            >
              HR-команда
            </button>
          </div>
          <form onSubmit={submit}>
            <label htmlFor="password">Пароль демо-аккаунта</label>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Введите пароль"
            />
            {error && <ErrorMessage message={error} />}
            <button className="button primary full" disabled={busy}>
              {busy ? 'Входим…' : 'Войти в Career Quest'}
              <ArrowRight size={18} />
            </button>
          </form>
          <p className="login-hint">
            Локальная демонстрация · синтетические данные.
            <br />
            Пароль вашей роли выдаёт участник команды.
          </p>
        </div>
      </div>
    </div>
  );
}
