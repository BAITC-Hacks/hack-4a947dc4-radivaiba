import { useState, type FormEvent } from 'react';
import { ArrowRight, Check, Leaf, ShieldCheck, Sparkles } from 'lucide-react';
import { api } from '../api/client';
import type { Session } from '../api/types';
import { LanguageSwitch, useI18n } from '../i18n';
import { Brand, ErrorMessage } from '../components/Shared';
import { TreeArt } from '../components/DevelopmentTree';
export default function Login({ onLogin }: { onLogin: (session: Session) => void }) {
  const { tr } = useI18n();
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError('');
    try {
      onLogin(await api.login(login.trim(), password));
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
          <span className="eyebrow">
            <Leaf size={14} />
            {tr(
              'ВАШЕ РАЗВИТИЕ. ВАШ ПУТЬ.',
              'СІЗДІҢ ДАМУЫҢЫЗ. СІЗДІҢ ЖОЛЫҢЫЗ.',
              'YOUR GROWTH. YOUR JOURNEY.',
            )}
          </span>
          <h1>
            {tr('Большой рост', 'Үлкен өсу', 'Meaningful growth')}
            <br />
            {tr('начинается', 'басталады', 'starts with')}
            <br />
            <em>{tr('с маленького шага.', 'шағын қадамнан.', 'one small step.')}</em>
          </h1>
          <p>
            {tr(
              'Найдите то, что полезно именно вам. Учитесь на практике, делитесь результатами и наблюдайте, как растёт ваше дерево.',
              'Өзіңізге пайдалы нәрсені табыңыз. Іс жүзінде үйреніңіз, нәтижелермен бөлісіңіз және ағашыңыздың өсуін бақылаңыз.',
              'Find what matters to you. Learn by doing, share your results and watch your own tree grow.',
            )}
          </p>
          <div className="intro-steps">
            <span>
              <Check size={15} />
              {tr('Ваша цель', 'Мақсатыңыз', 'Your goal')}
            </span>
            <i />
            <span>
              <Sparkles size={15} />
              {tr('Полезный шаг', 'Пайдалы қадам', 'A useful step')}
            </span>
            <i />
            <span>
              <Leaf size={15} />
              {tr('Настоящий рост', 'Нағыз өсу', 'Real growth')}
            </span>
          </div>
        </div>
        <div className="welcome-tree">
          <TreeArt id="welcome" />
        </div>
        <div className="story-footer">
          CAREER QUEST <span>HACKALEM AI · 2026</span>
        </div>
      </div>
      <div className="login-form-wrap">
        <div className="login-language">
          <LanguageSwitch />
        </div>
        <div className="login-form">
          <span className="eyebrow">
            {tr('РАДЫ ВАС ВИДЕТЬ', 'ҚОШ КЕЛДІҢІЗ', 'GOOD TO SEE YOU')}
          </span>
          <h2>{tr('Продолжим ваш путь', 'Жолыңызды жалғастырайық', 'Let’s keep growing')}</h2>
          <p>
            {tr(
              'Войдите в своё пространство развития.',
              'Өзіңіздің даму кеңістігіңізге кіріңіз.',
              'Sign in to your own development space.',
            )}
          </p>
          <form onSubmit={submit}>
            <label htmlFor="login">
              {tr('Логин', 'Логин', 'Login')}
              <input
                id="login"
                autoComplete="username"
                autoCapitalize="none"
                spellCheck={false}
                required
                value={login}
                onChange={(e) => setLogin(e.target.value)}
                placeholder={tr(
                  'Ваш логин сотрудника или HR',
                  'Қызметкер немесе HR логиніңіз',
                  'Your employee or HR login',
                )}
              />
            </label>
            <label htmlFor="password">
              {tr('Пароль', 'Құпиясөз', 'Password')}
              <input
                id="password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={tr('Введите пароль', 'Құпиясөзді енгізіңіз', 'Enter your password')}
              />
            </label>
            {error && <ErrorMessage message={error} />}
            <button className="button primary full" disabled={busy}>
              {busy
                ? tr('Входим…', 'Кірудеміз…', 'Signing in…')
                : tr('Войти в Career Quest', 'Career Quest-ке кіру', 'Sign in to Career Quest')}
              <ArrowRight size={17} />
            </button>
          </form>
          <p className="login-hint">
            <ShieldCheck size={13} />{' '}
            {tr(
              'У каждого сотрудника — свой профиль. Данные для входа выдаёт команда проекта.',
              'Әр қызметкердің өз профилі бар. Кіру деректерін жоба командасы береді.',
              'Every employee has their own profile. Your project team provides your login details.',
            )}
          </p>
        </div>
      </div>
    </div>
  );
}
