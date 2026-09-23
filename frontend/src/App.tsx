import { useCallback, useEffect, useState } from 'react';
import {
  ChevronRight,
  CircleHelp,
  Leaf,
  LogOut,
  ShieldCheck,
  TreeDeciduous,
  Trophy,
  Users,
} from 'lucide-react';
import { api, ApiError } from './api/client';
import type { EmployeeSummary, Session } from './api/types';
import { LanguageSwitch, useI18n } from './i18n';
import { Brand, ErrorMessage, Loading } from './components/Shared';
import Login from './pages/Login';
import Employee from './pages/Employee';
import HR from './pages/HR';
import Leaderboard from './pages/Leaderboard';
const currentPath = () => window.location.hash.replace(/^#/, '') || '/';
export default function App() {
  const { tr, locale, setLocale } = useI18n();
  const [session, setSession] = useState<Session | null>(null);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState('');
  const [path, setPath] = useState(currentPath);
  const [employees, setEmployees] = useState<EmployeeSummary[]>([]);
  const [showHelp, setShowHelp] = useState(false);
  useEffect(() => {
    api
      .session()
      .then((value) => {
        setSession(value);
        try {
          if (
            !localStorage.getItem('careerquest.locale') &&
            ['ru', 'kk', 'en'].includes(value.locale)
          )
            setLocale(value.locale);
        } catch {
          /* Keep current locale. */
        }
      })
      .catch((err) => {
        if (!(err instanceof ApiError && err.status === 401)) setError(err.message);
      })
      .finally(() => setReady(true));
    const hash = () => setPath(currentPath());
    const expired = () => {
      setSession(null);
      setEmployees([]);
    };
    window.addEventListener('hashchange', hash);
    window.addEventListener('session-expired', expired);
    return () => {
      window.removeEventListener('hashchange', hash);
      window.removeEventListener('session-expired', expired);
    };
  }, []);
  const loadEmployees = useCallback(async () => {
    try {
      setEmployees((await api.employees()).items || []);
    } catch (err) {
      setError((err as Error).message);
    }
  }, []);
  useEffect(() => {
    if (session?.role === 'hr') loadEmployees();
    else setEmployees([]);
  }, [session?.account_id, loadEmployees]);
  useEffect(() => {
    if (session && session.locale !== locale)
      api
        .preferences(locale)
        .then((value) => setSession(value))
        .catch((err) => setError(err.message));
  }, [locale, session?.account_id]);
  const navigate = (next: string) => {
    window.location.hash = next;
    setPath(next);
    window.scrollTo({ top: 0 });
  };
  const login = (value: Session) => {
    setSession(value);
    setError('');
    navigate(value.role === 'hr' ? '/hr' : `/employees/${value.employee_id}`);
  };
  async function logout() {
    try {
      await api.logout();
      setSession(null);
      setEmployees([]);
      setError('');
      navigate('/');
    } catch (err) {
      setError((err as Error).message);
    }
  }
  if (!ready) return <Loading />;
  if (!session)
    return (
      <>
        {error && (
          <div className="global-error">
            <ErrorMessage message={error} retry={() => window.location.reload()} />
          </div>
        )}
        <Login onLogin={login} />
      </>
    );
  const isHR = session.role === 'hr';
  const isLeaderboard = path === '/leaderboard';
  const isProfile = path.startsWith('/employees/') || (!isHR && !isLeaderboard);
  let requestedId = '';
  try {
    requestedId = decodeURIComponent(path.split('/')[2] || '');
  } catch {
    requestedId = '';
  }
  const selectedId = isHR ? requestedId : session.employee_id;
  const nav = isLeaderboard
    ? tr('Лидеры месяца', 'Ай көшбасшылары', 'Monthly leaders')
    : isProfile
      ? tr('Моя траектория', 'Менің даму жолым', 'My growth')
      : tr('Пространство HR', 'HR кеңістігі', 'HR workspace');
  return (
    <div className="app-shell">
      <header className="app-header">
        <Brand />
        <nav
          className="main-nav"
          aria-label={tr('Основная навигация', 'Негізгі навигация', 'Main navigation')}
        >
          {isHR ? (
            <button
              className={!isLeaderboard && !isProfile ? 'active' : ''}
              aria-current={!isLeaderboard && !isProfile ? 'page' : undefined}
              onClick={() => navigate('/hr')}
            >
              <Users size={18} />
              {tr('Пространство HR', 'HR кеңістігі', 'HR workspace')}
            </button>
          ) : (
            <button
              className={!isLeaderboard ? 'active' : ''}
              aria-current={!isLeaderboard ? 'page' : undefined}
              onClick={() => navigate(`/employees/${session.employee_id}`)}
            >
              <TreeDeciduous size={18} />
              {tr('Мой рост', 'Менің өсуім', 'My growth')}
            </button>
          )}
          <button
            className={isLeaderboard ? 'active' : ''}
            aria-current={isLeaderboard ? 'page' : undefined}
            onClick={() => navigate('/leaderboard')}
          >
            <Trophy size={17} />
            {tr('Лидеры месяца', 'Ай көшбасшылары', 'Monthly leaders')}
          </button>
        </nav>
        <div className="topbar-actions">
          <LanguageSwitch />
          <button
            className="icon-button"
            aria-label={tr('Как это работает', 'Бұл қалай жұмыс істейді', 'How it works')}
            aria-expanded={showHelp}
            aria-controls="help-panel"
            onClick={() => setShowHelp((v) => !v)}
          >
            <CircleHelp size={18} />
          </button>
          <span className="role-pill">
            <span className="role-avatar">
              {isHR ? <ShieldCheck size={17} /> : <Leaf size={18} />}
            </span>
            <span>
              {session.login}
              <small>{isHR ? 'HR' : tr('Сотрудник', 'Қызметкер', 'Employee')}</small>
            </span>
          </span>
          <button
            className="icon-button"
            aria-label={tr('Выйти', 'Шығу', 'Sign out')}
            onClick={logout}
          >
            <LogOut size={18} />
          </button>
        </div>
      </header>
      <main className="main-content">
        <div className="breadcrumb">
          Career Quest
          <ChevronRight size={13} />
          <span>{nav}</span>
        </div>
        {error && <ErrorMessage message={error} />}
        {showHelp && (
          <section className="help-panel panel" id="help-panel">
            <h2>
              {tr('Развитие, в котором есть смысл', 'Мағыналы даму', 'Growth with a purpose')}
            </h2>
            <p>
              {tr(
                'Выберите цель, откройте яблоко на дереве и выполните полезный модуль. Отправьте результат с доказательством. HR проверит его, после чего обновятся навыки, EXP и дерево.',
                'Мақсатты таңдап, ағаштағы алманы ашып, пайдалы модульді орындаңыз. Нәтижені дәлелмен жіберіңіз. HR тексергеннен кейін дағдылар, EXP және ағаш жаңартылады.',
                'Choose a goal, open an apple on your tree and complete a useful module. Submit your result with evidence. After HR approval, your skills, EXP and tree update.',
              )}
            </p>
            <p>
              {tr(
                'Яблоки — реальные модули. Энергия и уровень дерева отражают подтверждённый опыт. Навыки показывают готовность к цели; должностной грейд автоматически не меняется. Каждый месяц EXP считается заново, а дерево и общий опыт сохраняются.',
                'Алмалар — нақты модульдер. Ағаш қуаты мен деңгейі расталған тәжірибені көрсетеді. Дағдылар мақсатқа дайындықты көрсетеді; қызметтік деңгей автоматты өзгермейді. Ай сайын EXP жаңадан есептеледі, ал ағаш пен жалпы тәжірибе сақталады.',
                'Apples are real modules. Energy and tree level reflect approved experience. Skills indicate readiness for a goal; your job grade does not change automatically. Monthly EXP starts afresh, while your tree and lifetime experience remain.',
              )}
            </p>
            <button className="text-button" onClick={() => setShowHelp(false)}>
              {tr('Понятно', 'Түсінікті', 'Got it')}
            </button>
          </section>
        )}
        {isHR && isProfile && (
          <div className="employee-picker">
            <label htmlFor="employee-picker">
              {tr('Профиль сотрудника', 'Қызметкер профилі', 'Employee profile')}
            </label>
            <select
              id="employee-picker"
              value={selectedId}
              onChange={(e) => navigate(`/employees/${encodeURIComponent(e.target.value)}`)}
            >
              <option value="">
                {tr('Выберите сотрудника', 'Қызметкерді таңдаңыз', 'Select an employee')}
              </option>
              {employees.map((e) => (
                <option key={e.employee_id} value={e.employee_id}>
                  {e.full_name} · {e.employee_id}
                </option>
              ))}
            </select>
            <button className="text-button" onClick={() => navigate('/hr')}>
              {tr('К проверкам', 'Тексерулерге', 'Back to reviews')}
              <ArrowRightIcon />
            </button>
          </div>
        )}
        {isLeaderboard ? (
          <Leaderboard
            onEmployee={isHR ? (id) => navigate(`/employees/${encodeURIComponent(id)}`) : undefined}
          />
        ) : isProfile && selectedId ? (
          <Employee
            key={selectedId}
            id={selectedId}
            readOnly={isHR && session.employee_id !== selectedId}
          />
        ) : (
          <HR
            employees={employees}
            session={session}
            onEmployee={(id) => navigate(`/employees/${encodeURIComponent(id)}`)}
            onImport={loadEmployees}
          />
        )}
        <footer className="page-footer">
          <span>
            Career Quest ·{' '}
            {tr(
              'Расти в своём направлении.',
              'Өз бағытыңызда өсіңіз.',
              'Grow in your own direction.',
            )}
          </span>
          <span>HackAlem AI · 2026</span>
        </footer>
      </main>
    </div>
  );
}
function ArrowRightIcon() {
  return <ChevronRight size={14} />;
}
