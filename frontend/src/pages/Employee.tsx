import { useEffect, useRef, useState } from 'react';
import {
  ArrowRight,
  BookOpen,
  Check,
  CheckCircle2,
  ChevronDown,
  Clock3,
  Flag,
  GraduationCap,
  MapPin,
  Sparkles,
  Target,
  TrendingUp,
} from 'lucide-react';
import { api } from '../api/client';
import type { Profile, Recommendations, Recommendation } from '../api/types';
import {
  dateLabel,
  Empty,
  ErrorMessage,
  formatLabels,
  Loading,
  statusLabels,
  typeLabels,
} from '../components/Shared';

function RecommendationCard({
  item,
  index,
  busy,
  onComplete,
}: {
  item: Recommendation;
  index: number;
  busy: boolean;
  onComplete: () => void;
}) {
  return (
    <article className={`recommendation ${index === 0 ? 'featured' : ''}`}>
      <div className="rec-top">
        <span className="step-label">ШАГ {String(index + 1).padStart(2, '0')}</span>
        <span className="tag">{typeLabels[item.event.type] || item.event.type}</span>
      </div>
      <h3>{item.event.title}</h3>
      <p className="rec-description">{item.explanation}</p>
      <div className="event-meta">
        <span>
          <Clock3 size={14} />
          {item.event.duration_hours} ч
        </span>
        <span>
          <MapPin size={14} />
          {formatLabels[item.event.format]}
        </span>
      </div>
      {item.cross_role && <span className="tag amber">Подготовка к смене роли</span>}
      {item.in_progress && <span className="tag">Уже в процессе</span>}
      <div className="gain-list">
        {item.expected_gains.map((g) => (
          <div key={g.skill_id}>
            <span>{g.name || g.skill_id}</span>
            <strong>
              {g.before}
              <ArrowRight size={12} />
              {g.after}
            </strong>
          </div>
        ))}
      </div>
      <details className="evidence">
        <summary>
          Почему этот шаг
          <ChevronDown size={15} />
        </summary>
        <ul>
          {item.evidence.map((e) => (
            <li key={e.id}>{e.text}</li>
          ))}
        </ul>
      </details>
      <button
        className={`button ${index === 0 ? 'primary' : 'secondary'} full`}
        disabled={busy}
        onClick={onComplete}
      >
        <CheckCircle2 size={17} />
        {busy ? 'Сохраняем…' : 'Отметить выполнено'}
      </button>
    </article>
  );
}
export default function EmployeePage({ id }: { id: string }) {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [recommendations, setRecommendations] = useState<Recommendations | null>(null);
  const [error, setError] = useState('');
  const [recError, setRecError] = useState('');
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');
  const [tab, setTab] = useState<'skills' | 'history'>('skills');
  const [revision, setRevision] = useState(0);
  const requestGeneration = useRef(0);
  useEffect(() => {
    const abort = new AbortController();
    const generation = ++requestGeneration.current;
    setProfile(null);
    setRecommendations(null);
    setError('');
    setRecError('');
    setNotice('');
    api
      .profile(id, abort.signal)
      .then(setProfile)
      .catch((err) => {
        if (!abort.signal.aborted) setError(err.message);
      });
    api
      .recommendations(id, abort.signal)
      .then((value) => {
        if (generation === requestGeneration.current) setRecommendations(value);
      })
      .catch((err) => {
        if (!abort.signal.aborted) setRecError(err.message);
      });
    return () => {
      abort.abort();
      requestGeneration.current++;
    };
  }, [id, revision]);
  async function complete(item: Recommendation) {
    setBusy(true);
    setNotice('');
    setRecError('');
    const generation = ++requestGeneration.current;
    try {
      const result = await api.complete(id, item.event.event_id);
      if (generation !== requestGeneration.current) return;
      setProfile(result.profile);
      setNotice(
        result.changed
          ? `Готово! «${item.event.title}» завершено. Навыки и траектория обновлены.`
          : 'Эта активность уже учтена — прогресс не начисляется повторно.',
      );
      setRecommendations(null);
      const next = await api.recommendations(id);
      if (generation === requestGeneration.current) setRecommendations(next);
    } catch (err) {
      if (generation === requestGeneration.current) setRecError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  if (error) return <ErrorMessage message={error} retry={() => setRevision((v) => v + 1)} />;
  if (!profile) return <Loading />;
  const { employee, trajectory } = profile;
  const completed = profile.history.filter((h) => h.status === 'completed').length;
  const firstName = employee.full_name.split(' ')[0];
  return (
    <>
      <header className="page-heading">
        <div>
          <div className="eyebrow">ЛИЧНЫЙ КАБИНЕТ</div>
          <h1>
            Ваш следующий шаг, {firstName}
            <span className="heading-dot">.</span>
          </h1>
          <p>Каждая новая компетенция приближает вас к цели.</p>
        </div>
        <span className="snapshot">
          <span />
          Срез на {dateLabel(profile.as_of_date)}
        </span>
      </header>
      <div className="profile-grid">
        <section className="profile-card panel">
          <div className="avatar">
            {employee.full_name
              .split(' ')
              .map((s) => s[0])
              .slice(0, 2)
              .join('')}
          </div>
          <h2>{employee.full_name}</h2>
          <p>{employee.role}</p>
          <div className="profile-tags">
            <span className="tag">{employee.grade}</span>
            <span className="tag subtle">
              {{ office: 'В офисе', hybrid: 'Гибрид', remote: 'Удалённо' }[employee.work_format]}
            </span>
          </div>
          <div className="profile-details">
            <span>
              Подразделение<strong>{employee.department}</strong>
            </span>
            <span>
              В команде<strong>{employee.tenure_months} мес.</strong>
            </span>
            <span>
              Последняя оценка<strong>{dateLabel(employee.last_review_date)}</strong>
            </span>
          </div>
          <div className="completed-note">
            <GraduationCap size={19} />
            <strong>{completed}</strong> активностей завершено
          </div>
        </section>
        <section className="journey-card">
          <div className="journey-top">
            <span className="eyebrow light">
              <Flag size={15} /> ВАША ТРАЕКТОРИЯ
            </span>
            <span className="journey-chip">
              {trajectory.at_top_grade
                ? 'Углубление экспертизы'
                : trajectory.inferred
                  ? 'Предложенная цель'
                  : 'Карьерная цель'}
            </span>
          </div>
          <div className="journey-title">
            <div>
              <p>Двигаемся к</p>
              <h2>
                {trajectory.goal.target_grade}
                <br />
                <span>{trajectory.goal.target_role}</span>
              </h2>
            </div>
            <div
              className="progress-ring"
              style={{ '--progress': `${trajectory.progress}%` } as React.CSSProperties}
            >
              <div>
                <strong>
                  {trajectory.progress}
                  <small>%</small>
                </strong>
                <span>готовность</span>
              </div>
            </div>
          </div>
          <div className="journey-stages">
            <span>
              <Check size={16} />
              {employee.grade}
            </span>
            <i />
            <span className="active">
              <Target size={16} />
              Развитие навыков
            </span>
            <i />
            <span>
              <Flag size={16} />
              {trajectory.at_top_grade ? 'Экспертиза' : trajectory.goal.target_grade}
            </span>
          </div>
          <div className="journey-foot">
            <TrendingUp size={17} />
            {trajectory.critical_gaps > 0
              ? `${trajectory.critical_gaps} критических навыков требуют развития`
              : 'Критические требования цели закрыты'}
            <span>Прогресс по требованиям роли</span>
          </div>
        </section>
      </div>
      <section className="section recommendations-section">
        <div className="section-heading">
          <div>
            <span className="eyebrow">ПОДОБРАНО ДЛЯ ВАС</span>
            <h2>
              <Sparkles size={23} />С чего начать
            </h2>
          </div>
          <span className={`tag ${recommendations?.mode === 'llm' ? '' : 'subtle'}`}>
            {recommendations?.mode === 'llm' ? 'AI-рекомендации' : 'Объяснимый подбор'}
          </span>
        </div>
        {notice && (
          <div className="success" role="status">
            <CheckCircle2 size={19} />
            {notice}
          </div>
        )}
        {recError && <ErrorMessage message={recError} retry={() => setRevision((v) => v + 1)} />}
        {!recommendations && !recError && (
          <Loading text="Подбираем полезные шаги — до 10 секунд…" />
        )}
        {recommendations && (
          <>
            {recommendations.items.length ? (
              <div className="recommendation-grid">
                {recommendations.items.map((item, index) => (
                  <RecommendationCard
                    key={item.event.event_id}
                    item={item}
                    index={index}
                    busy={busy}
                    onComplete={() => complete(item)}
                  />
                ))}
              </div>
            ) : (
              <Empty title="Следующий шаг требует внимания" text={recommendations.empty_reason} />
            )}
            <p className="section-note">
              {recommendations.notice} Завершение — демо-действие на дату среза.
            </p>
          </>
        )}
      </section>
      <section className="section panel skills-panel">
        <div className="tabs">
          <button className={tab === 'skills' ? 'active' : ''} onClick={() => setTab('skills')}>
            <BookOpen size={17} />
            Карта навыков<span>{trajectory.skills.length}</span>
          </button>
          <button className={tab === 'history' ? 'active' : ''} onClick={() => setTab('history')}>
            <Clock3 size={17} />
            История развития<span>{profile.history.length}</span>
          </button>
        </div>
        {tab === 'skills' ? (
          <>
            <div className="table-intro">
              <p>
                Ваши навыки относительно цели <strong>{trajectory.goal.target_grade}</strong>
              </p>
              <span>
                <i className="critical-dot" />
                Критично для перехода
              </span>
            </div>
            <div className="table-scroll">
              <table className="skills-table">
                <thead>
                  <tr>
                    <th>Компетенция</th>
                    <th>Тип</th>
                    <th>Уровень навыка</th>
                    <th>Сейчас / цель</th>
                    <th>До цели</th>
                  </tr>
                </thead>
                <tbody>
                  {trajectory.skills.map((s) => (
                    <tr key={s.skill_id}>
                      <td>
                        <span className="skill-name">
                          {s.critical && <i className="critical-dot" />}
                          {s.name}
                        </span>
                      </td>
                      <td>
                        <span className="quiet-tag">{s.type}</span>
                      </td>
                      <td>
                        <div className="skill-blocks" aria-label={`Уровень ${s.current} из 5`}>
                          {[1, 2, 3, 4, 5].map((n) => (
                            <span
                              key={n}
                              className={
                                n <= s.current ? 'filled' : n <= s.required ? 'needed' : ''
                              }
                            />
                          ))}
                        </div>
                      </td>
                      <td>
                        <strong>{s.current}</strong>
                        <span className="muted"> / {s.required || '—'}</span>
                      </td>
                      <td>
                        {s.gap > 0 ? (
                          <span className="gap-tag">+{s.gap} ур.</span>
                        ) : (
                          <span className="skill-done">
                            <Check size={16} />
                            {s.required ? 'Готово' : 'Дополнительно'}
                          </span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="section-note">
              Шкала 0–5. Прогресс учитывает завершённое обучение после последней оценки; критические
              навыки имеют вес 3.
            </p>
          </>
        ) : (
          <div className="history-list">
            {profile.history.length ? (
              profile.history.map((h) => (
                <div className="history-item" key={h.record_id}>
                  <span className={`history-icon ${h.status === 'completed' ? 'done' : ''}`}>
                    {h.status === 'completed' ? <Check size={18} /> : <Clock3 size={18} />}
                  </span>
                  <div>
                    <strong>{h.event_title}</strong>
                    <small>
                      {dateLabel(h.date)}
                      {h.demo ? ' · демо-завершение' : ''}
                    </small>
                  </div>
                  <span className={`tag ${h.status === 'completed' ? '' : 'subtle'}`}>
                    {statusLabels[h.status]}
                  </span>
                </div>
              ))
            ) : (
              <Empty title="История пока пуста" text="Завершённые активности появятся здесь." />
            )}
          </div>
        )}
      </section>
    </>
  );
}
