import { useCallback, useEffect, useRef, useState } from 'react';
import {
  ArrowRight,
  BookOpen,
  Check,
  CheckCircle2,
  ChevronDown,
  Clock3,
  Flag,
  Leaf,
  Pencil,
  ShieldCheck,
  Sparkles,
  Target,
  Trophy,
  Zap,
} from 'lucide-react';
import { api } from '../api/client';
import type { Development, Goal, ModuleView, Profile, Recommendations } from '../api/types';
import { formatDate, useI18n, useLabels } from '../i18n';
import DevelopmentTree from '../components/DevelopmentTree';
import ModulePanel from '../components/ModulePanel';
import { Empty, ErrorMessage, Loading, Modal, Status } from '../components/Shared';
function GoalEditor({
  profile,
  onSaved,
  onClose,
}: {
  profile: Profile;
  onSaved: (profile: Profile) => void;
  onClose: () => void;
}) {
  const { tr } = useI18n();
  const [goals, setGoals] = useState<Goal[] | null>(null);
  const [selected, setSelected] = useState(
    profile.employee.career_goal
      ? `${profile.employee.career_goal.target_role}|${profile.employee.career_goal.target_grade}`
      : '',
  );
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    api
      .goals()
      .then((result) => setGoals(result.items))
      .catch((err) => setError(err.message));
  }, []);
  async function save() {
    setBusy(true);
    setError('');
    const value = goals?.find((g) => `${g.target_role}|${g.target_grade}` === selected) || null;
    try {
      onSaved(await api.goal(profile.employee.employee_id, value));
      onClose();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      title={tr(
        'Куда вы хотите вырасти?',
        'Қай бағытта өскіңіз келеді?',
        'Where would you like to grow?',
      )}
      onClose={onClose}
    >
      <p>
        {tr(
          'Выберите роль и уровень. Мы пересчитаем нужные навыки и предложим подходящие шаги. История и EXP сохранятся.',
          'Рөл мен деңгейді таңдаңыз. Қажетті дағдыларды қайта есептеп, қолайлы қадамдарды ұсынамыз. Тарих пен EXP сақталады.',
          'Choose a role and level. We will update the skills and next steps you need. Your history and EXP stay with you.',
        )}
      </p>
      {error && <ErrorMessage message={error} />}{' '}
      {!goals ? (
        <Loading />
      ) : (
        <div className="dialog-section">
          <label>
            {tr('Карьерная цель', 'Мансаптық мақсат', 'Career goal')}
            <select value={selected} onChange={(e) => setSelected(e.target.value)}>
              <option value="">
                {tr(
                  'Предложить следующую ступень',
                  'Келесі деңгейді ұсыну',
                  'Suggest my next level',
                )}
              </option>
              {goals.map((goal) => (
                <option
                  key={`${goal.target_role}|${goal.target_grade}`}
                  value={`${goal.target_role}|${goal.target_grade}`}
                >
                  {goal.target_grade} · {goal.target_role}
                </option>
              ))}
            </select>
          </label>
          <div className="form-actions">
            <button className="button secondary" onClick={onClose}>
              {tr('Отмена', 'Бас тарту', 'Cancel')}
            </button>
            <button className="button primary" disabled={busy} onClick={save}>
              {tr('Сохранить цель', 'Мақсатты сақтау', 'Save goal')}
              <Check size={16} />
            </button>
          </div>
        </div>
      )}
    </Modal>
  );
}
export default function EmployeePage({ id, readOnly = false }: { id: string; readOnly?: boolean }) {
  const { tr, locale } = useI18n();
  const labels = useLabels();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [development, setDevelopment] = useState<Development | null>(null);
  const [recommendations, setRecommendations] = useState<Recommendations | null>(null);
  const [error, setError] = useState('');
  const [recError, setRecError] = useState('');
  const [selected, setSelected] = useState('');
  const [showGoal, setShowGoal] = useState(false);
  const [notice, setNotice] = useState('');
  const [celebrate, setCelebrate] = useState(false);
  const generation = useRef(0);
  const seenAwards = useRef<Set<string> | null>(null);
  const [celebrateBranch, setCelebrateBranch] = useState('');
  const refresh = useCallback(
    async (quiet = false) => {
      const seq = ++generation.current;
      if (!quiet) setError('');
      try {
        const [p, d] = await Promise.all([api.profile(id), api.development(id)]);
        if (generation.current !== seq) return;
        setProfile(p);
        setDevelopment(d);
        setError('');
      } catch (err) {
        if (generation.current === seq && !quiet) setError((err as Error).message);
      }
    },
    [id, locale],
  );
  useEffect(() => {
    refresh();
    return () => {
      generation.current++;
    };
  }, [refresh]);
  useEffect(() => {
    if (!development) return;
    const abort = new AbortController();
    setRecommendations(null);
    setRecError('');
    api
      .recommendations(id, abort.signal)
      .then(setRecommendations)
      .catch((err) => {
        if (!abort.signal.aborted) setRecError(err.message);
      });
    return () => abort.abort();
  }, [id, locale, development?.revision, development?.business_date]);
  const pending = development?.items.some((m) => m.state === 'pending');
  useEffect(() => {
    const focus = () => {
      if (document.visibilityState === 'visible') refresh(true);
    };
    window.addEventListener('focus', focus);
    document.addEventListener('visibilitychange', focus);
    const timer = pending ? window.setInterval(focus, 10000) : undefined;
    return () => {
      window.removeEventListener('focus', focus);
      document.removeEventListener('visibilitychange', focus);
      if (timer) clearInterval(timer);
    };
  }, [pending, refresh]);
  useEffect(() => {
    if (!development) return;
    const awards = (development.experience.entries || []).filter((entry) => entry.amount > 0);
    const added = seenAwards.current && awards.find((entry) => !seenAwards.current!.has(entry.id));
    if (added) {
      setCelebrateBranch(
        development.items.find((module) => module.event.event_id === added.event_id)?.branch ||
          'professional',
      );
      setCelebrate(true);
      setNotice(
        tr(
          'HR подтвердил результат — ваше дерево подросло!',
          'HR нәтижені растады — ағашыңыз өсті!',
          'HR approved your result — your tree has grown!',
        ),
      );
    }
    seenAwards.current = new Set(awards.map((entry) => entry.id));
  }, [development?.revision]);
  useEffect(() => {
    if (!celebrate) return;
    const timer = window.setTimeout(() => setCelebrate(false), 1500);
    return () => clearTimeout(timer);
  }, [celebrate]);
  if (error && !profile) return <ErrorMessage message={error} retry={() => refresh()} />;
  if (!profile || !development) return <Loading />;
  const { employee, trajectory } = profile;
  const modules = development.items || [];
  const xp = development.experience;
  const recommendationMap = new Map(
    (recommendations?.items || []).map((item) => [item.event.event_id, item]),
  );
  const active =
    modules.find((m) => m.state === 'changes_requested') ||
    modules.find((m) => m.state === 'in_progress');
  const recommended = (recommendations?.items || [])
    .map((item) => modules.find((m) => m.event.event_id === item.event.event_id))
    .filter(
      (m): m is ModuleView =>
        !!m && m.state !== 'completed' && m.state !== 'locked' && m.state !== 'pending',
    );
  const primary =
    active ||
    recommended[0] ||
    modules.find((m) => m.recommended && m.state === 'available') ||
    modules.find((m) => m.state === 'pending');
  const primaryRecommendation = primary ? recommendationMap.get(primary.event.event_id) : undefined;
  const alternatives = [
    ...recommended,
    ...modules.filter((m) => m.recommended && m.state === 'available'),
  ]
    .filter(
      (m, i, all) =>
        m !== primary && all.findIndex((other) => other.event.event_id === m.event.event_id) === i,
    )
    .slice(0, 2);
  const opened = modules.find((m) => m.event.event_id === selected);
  const pendingModules = modules.filter((m) =>
    ['pending', 'changes_requested', 'in_progress'].includes(m.state),
  );
  const openModule = (m: ModuleView) => setSelected(m.event.event_id);
  const title = employee.full_name.split(' ')[0];
  return (
    <>
      <header className="page-heading">
        <div>
          <span className="eyebrow">
            <Leaf size={13} />
            {tr(
              'МАЛЕНЬКИЕ ШАГИ. БОЛЬШОЙ РОСТ.',
              'ШАҒЫН ҚАДАМДАР. ҮЛКЕН ӨСУ.',
              'SMALL STEPS. MEANINGFUL GROWTH.',
            )}
          </span>
          <h1>
            {readOnly
              ? employee.full_name
              : `${tr('Растём дальше,', 'Әрі қарай өсейік,', 'Keep growing,')} ${title}`}
            <span className="heading-dot">.</span>
          </h1>
          <p>
            {tr(
              'Ваш опыт — корни. Следующий шаг — новая ветка.',
              'Тәжірибеңіз — тамыр. Келесі қадам — жаңа бұтақ.',
              'Your experience is the foundation. Your next step is a new branch.',
            )}
          </p>
        </div>
        <div className="identity">
          <span className="avatar">
            {employee.full_name
              .split(' ')
              .slice(0, 2)
              .map((part) => part[0])
              .join('')}
          </span>
          <div>
            <strong>{employee.full_name}</strong>
            <p>
              {employee.grade} {employee.role}
            </p>
            <p>{employee.department}</p>
          </div>
        </div>
      </header>
      {error && <ErrorMessage message={error} retry={() => refresh()} />}{' '}
      {notice && (
        <div className="success" role="status">
          <CheckCircle2 size={19} />
          {notice}
        </div>
      )}
      <section className="goal-card panel">
        <div>
          <div className="row space-between">
            <span className="eyebrow">
              <Flag size={13} />
              {trajectory.inferred
                ? tr('ПРЕДЛОЖЕННАЯ ЦЕЛЬ', 'ҰСЫНЫЛҒАН МАҚСАТ', 'SUGGESTED GOAL')
                : tr('МОЯ КАРЬЕРНАЯ ЦЕЛЬ', 'МЕНІҢ МАНСАПТЫҚ МАҚСАТЫМ', 'MY CAREER GOAL')}
            </span>
            {!readOnly && (
              <button className="text-button" onClick={() => setShowGoal(true)}>
                <Pencil size={13} />
                {tr('Изменить', 'Өзгерту', 'Change')}
              </button>
            )}
          </div>
          <h2>
            {trajectory.goal.target_grade} <span className="muted">·</span>{' '}
            {trajectory.goal.target_role}
          </h2>
          <p className="goal-current">
            {trajectory.at_top_grade
              ? tr(
                  'Развиваем экспертизу в текущей роли.',
                  'Қазіргі рөлдегі тәжірибені дамытамыз.',
                  'Deepening expertise in your current role.',
                )
              : `${tr('Сейчас:', 'Қазір:', 'Today:')} ${employee.grade} ${employee.role}`}
            {trajectory.critical_gaps > 0 &&
              ` · ${tr('В фокусе навыков:', 'Назардағы дағдылар:', 'Priority skill gaps:')} ${trajectory.critical_gaps}`}
          </p>
        </div>
        <div className="goal-right">
          <div className="row">
            <span>{tr('Требования цели', 'Мақсат талаптары', 'Goal requirements')}</span>
            <strong>{trajectory.progress}%</strong>
          </div>
          <div
            className="progress-track"
            role="progressbar"
            aria-valuenow={trajectory.progress}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-label={tr('Готовность к цели', 'Мақсатқа дайындық', 'Readiness for your goal')}
          >
            <span style={{ width: `${trajectory.progress}%` }} />
          </div>
          <small className="muted">
            {tr(
              'Показатель навыков, не обещание повышения',
              'Дағды көрсеткіші, жоғарылау кепілі емес',
              'A skills indicator, not a promise of promotion',
            )}
          </small>
        </div>
      </section>
      <div className="growth-layout">
        <DevelopmentTree
          modules={modules}
          experience={xp}
          onOpen={openModule}
          celebrate={celebrate}
          celebrateBranch={celebrateBranch}
        />
        <section className="next-step panel">
          <span className="eyebrow">
            <Sparkles size={14} />
            {tr('СЛЕДУЮЩИЙ ШАГ ДЛЯ ВАС', 'СІЗ ҮШІН КЕЛЕСІ ҚАДАМ', 'YOUR NEXT GOOD STEP')}
          </span>
          {primary ? (
            <>
              <div className="row wrap">
                <Status value={primary.state} />
                <span className="tag subtle">
                  {labels.category(primary.category || primary.branch)}
                </span>
              </div>
              <h2>{primary.event.title}</h2>
              <p className="benefit">
                {recommendationMap.get(primary.event.event_id)?.explanation ||
                  primary.config.outcome[locale] ||
                  primary.config.outcome.ru}
              </p>
              <div className="module-meta">
                <span>
                  <Clock3 size={14} />
                  {primary.event.duration_hours} {tr('ч', 'сағ', 'h')}
                </span>
                <span>{labels.format(primary.event.format)}</span>
                <span>
                  <Zap size={14} />
                  {primary.config.reward_exp} EXP
                </span>
              </div>
              <div className="gain-pills">
                {(primary.expected_gains || []).slice(0, 3).map((gain) => (
                  <span className="gain-pill" key={gain.skill_id}>
                    {gain.name}
                    <strong>
                      {gain.before} → {gain.after}
                    </strong>
                  </span>
                ))}
              </div>
              {!!primaryRecommendation?.evidence?.length && (
                <details className="recommendation-facts">
                  <summary>
                    <span>
                      <Sparkles size={15} />
                      {tr('Почему этот шаг', 'Неліктен бұл қадам', 'Why this step')}
                    </span>
                    <ChevronDown size={15} />
                  </summary>
                  <ul>
                    {primaryRecommendation.evidence.map((fact) => (
                      <li key={fact.id}>{fact.text}</li>
                    ))}
                  </ul>
                </details>
              )}
              <button className="button primary full" onClick={() => openModule(primary)}>
                {primary.state === 'pending'
                  ? tr('Посмотреть заявку', 'Өтінімді қарау', 'View submission')
                  : primary.state === 'changes_requested'
                    ? tr('Доработать результат', 'Нәтижені толықтыру', 'Improve your result')
                    : primary.state === 'in_progress'
                      ? tr('Продолжить модуль', 'Модульді жалғастыру', 'Continue module')
                      : tr('Посмотреть мой шаг', 'Менің қадамымды қарау', 'Explore my next step')}
                <ArrowRight size={17} />
              </button>
              <p className="next-step-foot">
                <ShieldCheck size={12} />{' '}
                {tr(
                  'Один полезный результат. Подтверждение HR.',
                  'Бір пайдалы нәтиже. HR растауы.',
                  'One meaningful result. Approved by HR.',
                )}
              </p>
            </>
          ) : (
            <Empty
              title={tr(
                'Выберите новое направление',
                'Жаңа бағытты таңдаңыз',
                'Choose your next direction',
              )}
              text={
                recommendations?.empty_reason ||
                tr(
                  'Для этой цели пока нет подходящего шага. Посмотрите дерево или обсудите цель с HR.',
                  'Бұл мақсатқа қолайлы қадам әзірге жоқ. Ағашты қараңыз немесе HR-мен мақсатты талқылаңыз.',
                  'There is no suitable next step for this goal yet. Explore the tree or discuss your goal with HR.',
                )
              }
            />
          )}
          {recError && (
            <p className="section-note">
              {tr(
                'Персональное объяснение временно недоступно. Модули работают.',
                'Жеке түсіндірме уақытша қолжетімсіз. Модульдер жұмыс істейді.',
                'Personal explanations are temporarily unavailable. Modules still work.',
              )}
            </p>
          )}
          {recommendations && (
            <p className="section-note">
              {recommendations.mode === 'llm'
                ? tr('Подобрано с AI', 'AI көмегімен таңдалды', 'Selected with AI')
                : tr(
                    'Подбор по навыкам · резервный режим AI',
                    'Дағдылар бойынша таңдау · AI резервтік режимі',
                    'Skills-based suggestions · AI fallback',
                  )}
            </p>
          )}
        </section>
      </div>
      {alternatives.length > 0 && (
        <section className="section">
          <div className="section-heading">
            <h2>
              {tr(
                'Можно пойти другим путём',
                'Басқа жолды таңдауға болады',
                'There is more than one way to grow',
              )}
            </h2>
            <span className="muted small">
              {tr('Выбор за вами', 'Таңдау сізде', 'Your choice')}
            </span>
          </div>
          <div className="alternatives">
            {alternatives.map((module) => (
              <article className="alternative panel" key={module.event.event_id}>
                <span className="alternative-icon">
                  <BookOpen size={20} />
                </span>
                <div className="alternative-content">
                  <h3>{module.event.title}</h3>
                  <p>{labels.category(module.category || module.branch)}</p>
                  <div className="module-meta">
                    <span>
                      <Clock3 size={13} />
                      {module.event.duration_hours} {tr('ч', 'сағ', 'h')}
                    </span>
                    <span>
                      <Zap size={13} />
                      {module.config.reward_exp} EXP
                    </span>
                  </div>
                  <button
                    className="text-button"
                    aria-label={`${tr('Подробнее', 'Толығырақ', 'Explore')} ${module.event.title}`}
                    onClick={() => openModule(module)}
                  >
                    {tr('Подробнее', 'Толығырақ', 'Explore')}
                    <ArrowRight size={14} />
                  </button>
                </div>
              </article>
            ))}
          </div>
        </section>
      )}
      <section className="experience-strip panel">
        <div>
          <span className="exp-label">
            <Zap size={14} />
            {tr('EXP ЗА МЕСЯЦ', 'АЙЛЫҚ EXP', 'MONTHLY EXP')}
          </span>
          <strong>{xp.monthly_exp}</strong>
          <small>{xp.month}</small>
        </div>
        <div>
          <span className="exp-label">
            <SproutIcon />
            {tr('ОБЩИЙ ОПЫТ', 'ЖАЛПЫ ТӘЖІРИБЕ', 'LIFETIME EXPERIENCE')}
          </span>
          <strong>{xp.total_exp}</strong>
          <small>
            {tr('Сохраняется каждый месяц', 'Ай сайын сақталады', 'Grows across months')}
          </small>
        </div>
        <div>
          <span className="exp-label">
            <Trophy size={14} />
            {tr(
              'ВАШ ПРОГРЕСС ИМЕЕТ ЗНАЧЕНИЕ',
              'СІЗДІҢ ПРОГРЕСІҢІЗ МАҢЫЗДЫ',
              'YOUR PROGRESS MATTERS',
            )}
          </span>
          <p>
            {tr(
              'Дерево растёт за подтверждённые результаты. Новый месяц — новая возможность, без потери опыта.',
              'Ағаш расталған нәтижелер арқылы өседі. Жаңа ай — тәжірибені жоғалтпайтын жаңа мүмкіндік.',
              'Your tree grows with approved results. A new month is a fresh opportunity, with no lost experience.',
            )}
          </p>
        </div>
      </section>
      {pendingModules.length > 0 && (
        <details className="details-section panel" open>
          <summary>
            <span>
              <Clock3 size={18} />
              {tr('Мои активные модули', 'Менің белсенді модульдерім', 'My active modules')}{' '}
              <span className="tag">{pendingModules.length}</span>
            </span>
            <ChevronDown size={17} />
          </summary>
          <div className="details-body history-list">
            {pendingModules.map((module) => (
              <div className="history-item" key={module.event.event_id}>
                <span className="history-icon">
                  <Clock3 size={17} />
                </span>
                <div>
                  <strong>{module.event.title}</strong>
                  <small>
                    {module.config.reward_exp} EXP ·{' '}
                    {labels.category(module.category || module.branch)}
                  </small>
                </div>
                <Status value={module.state} />
                <button
                  className="icon-button"
                  onClick={() => openModule(module)}
                  aria-label={`${tr('Открыть', 'Ашу', 'Open')} ${module.event.title}`}
                >
                  <ArrowRight size={17} />
                </button>
              </div>
            ))}
          </div>
        </details>
      )}
      <details className="details-section panel">
        <summary>
          <span>
            <Target size={18} />
            {tr(
              'Навыки и требования цели',
              'Дағдылар және мақсат талаптары',
              'Skills and goal requirements',
            )}
          </span>
          <ChevronDown size={17} />
        </summary>
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>{tr('Навык', 'Дағды', 'Skill')}</th>
                <th>{tr('Сейчас', 'Қазір', 'Current')}</th>
                <th>{tr('Для цели', 'Мақсат үшін', 'Required')}</th>
                <th>{tr('Осталось', 'Қалды', 'Remaining')}</th>
              </tr>
            </thead>
            <tbody>
              {trajectory.skills.map((skill) => (
                <tr key={skill.skill_id}>
                  <td>
                    {skill.critical && <i className="critical-dot" />}
                    {skill.name}
                  </td>
                  <td>
                    <div className="row">
                      <div className="skill-blocks" aria-hidden="true">
                        {[1, 2, 3, 4, 5].map((n) => (
                          <span
                            key={n}
                            className={
                              n <= skill.current ? 'filled' : n <= skill.required ? 'needed' : ''
                            }
                          />
                        ))}
                      </div>
                      {skill.current}/5
                    </div>
                  </td>
                  <td>{skill.required || '—'}</td>
                  <td>
                    {skill.gap > 0 ? (
                      <span className="gap-tag">+{skill.gap}</span>
                    ) : (
                      <span className="skill-done">
                        <Check size={16} />
                        {tr('Готово', 'Дайын', 'Met')}
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="details-body section-note">
          {tr(
            'Жёлтая точка — навык, особенно важный для цели. Показатель учитывает подтверждённые результаты после последней оценки.',
            'Сары нүкте — мақсат үшін ерекше маңызды дағды. Көрсеткіш соңғы бағалаудан кейінгі расталған нәтижелерді ескереді.',
            'A gold dot marks a skill especially important for your goal. Progress includes approved results since your last review.',
          )}
        </p>
      </details>
      <details className="details-section panel">
        <summary>
          <span>
            <BookOpen size={18} />
            {tr('История развития', 'Даму тарихы', 'Development history')}
          </span>
          <ChevronDown size={17} />
        </summary>
        <div className="details-body history-list">
          {profile.history?.length ? (
            [...profile.history].reverse().map((item) => (
              <div className="history-item" key={item.record_id}>
                <span className="history-icon">
                  {item.status === 'completed' ? <Check size={17} /> : <Clock3 size={17} />}
                </span>
                <div>
                  <strong>{item.event_title}</strong>
                  <small>{formatDate(item.date, locale)}</small>
                </div>
                <Status value={item.status} />
              </div>
            ))
          ) : (
            <Empty
              title={tr('История ещё впереди', 'Тарих әлі алда', 'Your story starts here')}
              text={tr(
                'Здесь появятся ваши результаты.',
                'Нәтижелеріңіз осында пайда болады.',
                'Your results will appear here.',
              )}
            />
          )}
        </div>
      </details>
      <details className="details-section panel">
        <summary>
          <span>
            <Zap size={18} />
            {tr('История EXP', 'EXP тарихы', 'EXP history')}
          </span>
          <ChevronDown size={17} />
        </summary>
        <div className="details-body history-list exp-ledger">
          {xp.entries?.length ? (
            [...xp.entries].reverse().map((entry) => (
              <div className="history-item" key={entry.id}>
                <div>
                  <strong>
                    {modules.find((m) => m.event.event_id === entry.event_id)?.event.title ||
                      entry.event_id}
                  </strong>
                  <small>
                    {formatDate(entry.recorded_at, locale, true)} · {entry.month}
                    {entry.reversal_of
                      ? ` · ${tr('Отмена начисления', 'Есептеуді жою', 'Award reversal')}`
                      : ''}
                  </small>
                </div>
                <span className={`amount ${entry.amount < 0 ? 'negative' : ''}`}>
                  {entry.amount > 0 ? '+' : ''}
                  {entry.amount} EXP
                </span>
              </div>
            ))
          ) : (
            <Empty
              title={tr('Первый EXP — впереди', 'Алғашқы EXP алда', 'Your first EXP is ahead')}
              text={tr(
                'EXP появится после подтверждения результата HR. Импортированная история не начисляет EXP.',
                'EXP нәтижені HR растағаннан кейін пайда болады. Импортталған тарих EXP бермейді.',
                'EXP appears after HR approves a result. Imported history does not award EXP.',
              )}
            />
          )}
        </div>
      </details>
      <p className="section-note">
        {tr('Дата демонстрации:', 'Демонстрация күні:', 'Demo date:')}{' '}
        {formatDate(development.business_date, locale)} ·{' '}
        {tr('Последняя оценка:', 'Соңғы бағалау:', 'Last review:')}{' '}
        {formatDate(employee.last_review_date, locale)}
      </p>
      {showGoal && (
        <GoalEditor
          profile={profile}
          onSaved={(value) => {
            setProfile(value);
            refresh();
          }}
          onClose={() => setShowGoal(false)}
        />
      )}
      {opened && (
        <ModulePanel
          key={opened.event.event_id}
          module={opened}
          employeeId={id}
          readOnly={readOnly}
          onClose={() => setSelected('')}
          onChanged={() => refresh(true)}
          onAlternative={(eventId) => {
            if (modules.some((m) => m.event.event_id === eventId)) setSelected(eventId);
          }}
        />
      )}
    </>
  );
}
function SproutIcon() {
  return <Leaf size={14} />;
}
