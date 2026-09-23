import { useEffect, useState, type FormEvent } from 'react';
import {
  ArrowRight,
  BarChart3,
  CheckCircle2,
  ClipboardCheck,
  FileUp,
  Search,
  Users,
  X,
} from 'lucide-react';
import { api, ApiError } from '../api/client';
import type { EmployeeSummary, Overview, Session } from '../api/types';
import { formatDate, useI18n, useLabels } from '../i18n';
import { Empty, ErrorMessage, Loading, Stat } from '../components/Shared';
import ReviewQueue from '../components/ReviewQueue';
interface ImportIssue {
  file: string;
  record: string;
  reason: string;
}
export default function HR({
  employees,
  session,
  onEmployee,
  onImport,
}: {
  employees: EmployeeSummary[];
  session: Session;
  onEmployee: (id: string) => void;
  onImport: () => void;
}) {
  const { tr, locale } = useI18n();
  const labels = useLabels();
  const [tab, setTab] = useState<'reviews' | 'employees' | 'analytics'>('reviews');
  const [overview, setOverview] = useState<Overview | null>(null);
  const [error, setError] = useState('');
  const [query, setQuery] = useState('');
  const [department, setDepartment] = useState('');
  const [role, setRole] = useState('');
  const [grade, setGrade] = useState('');
  const [showImport, setShowImport] = useState(false);
  const [busy, setBusy] = useState(false);
  const [importError, setImportError] = useState('');
  const [importDetails, setImportDetails] = useState<ImportIssue[]>([]);
  const [notice, setNotice] = useState('');
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    const abort = new AbortController();
    setError('');
    api
      .overview(abort.signal)
      .then(setOverview)
      .catch((err) => {
        if (!abort.signal.aborted) setError(err.message);
      });
    return () => abort.abort();
  }, [revision, locale, tab]);
  async function upload(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    for (const key of ['employees', 'history']) {
      const file = data.get(key);
      if (file instanceof File && !file.size) data.delete(key);
    }
    if (![...data.keys()].length) {
      setImportError(
        tr(
          'Выберите хотя бы один файл.',
          'Кем дегенде бір файлды таңдаңыз.',
          'Choose at least one file.',
        ),
      );
      return;
    }
    setBusy(true);
    setImportError('');
    setImportDetails([]);
    setNotice('');
    try {
      const result = await api.import(data);
      setNotice(
        `${tr('Импорт завершён.', 'Импорт аяқталды.', 'Import complete.')} ${tr('Сотрудники: добавлено', 'Қызметкерлер: қосылды', 'Employees added:')} ${result.employees_added}, ${tr('обновлено', 'жаңартылды', 'updated:')} ${result.employees_updated}. ${tr('История: добавлено', 'Тарих: қосылды', 'History added:')} ${result.history_added}, ${tr('обновлено', 'жаңартылды', 'updated:')} ${result.history_updated}.`,
      );
      form.reset();
      setShowImport(false);
      setRevision((v) => v + 1);
      onImport();
    } catch (err) {
      setImportError((err as Error).message);
      if (err instanceof ApiError && Array.isArray(err.details)) {
        setImportDetails(
          err.details.filter(
            (issue): issue is ImportIssue =>
              !!issue &&
              typeof issue === 'object' &&
              typeof issue.file === 'string' &&
              typeof issue.reason === 'string',
          ),
        );
      }
    } finally {
      setBusy(false);
    }
  }
  const filtered = employees.filter(
    (e) =>
      `${e.full_name} ${e.employee_id} ${e.role} ${e.department}`
        .toLowerCase()
        .includes(query.toLowerCase()) &&
      (!department || e.department === department) &&
      (!role || e.role === role) &&
      (!grade || e.grade === grade),
  );
  return (
    <>
      <header className="page-heading">
        <div>
          <span className="eyebrow">
            <Users size={13} />
            {tr('ПРОСТРАНСТВО HR', 'HR КЕҢІСТІГІ', 'HR WORKSPACE')}
          </span>
          <h1>
            {tr('Помогайте людям расти', 'Адамдардың өсуіне көмектесіңіз', 'Help people grow')}
            <span className="heading-dot">.</span>
          </h1>
          <p>
            {tr(
              'Замечайте усилия. Подтверждайте результаты. Открывайте возможности.',
              'Талпынысты байқаңыз. Нәтижені растаңыз. Мүмкіндіктер ашыңыз.',
              'Recognise effort. Approve results. Open up opportunities.',
            )}
          </p>
        </div>
        <button className="button secondary" onClick={() => setShowImport((value) => !value)}>
          <FileUp size={17} />
          {tr('Импорт данных', 'Деректер импорты', 'Import data')}
        </button>
      </header>
      {notice && (
        <div className="success" role="status">
          <CheckCircle2 size={19} />
          {notice}
        </div>
      )}
      {showImport && (
        <section className="panel import-panel">
          <div className="section-heading">
            <h2>
              {tr(
                'Обновить сотрудников и историю',
                'Қызметкерлер мен тарихты жаңарту',
                'Update employees and history',
              )}
            </h2>
            <button
              className="icon-button"
              onClick={() => setShowImport(false)}
              aria-label={tr('Закрыть импорт', 'Импортты жабу', 'Close import')}
            >
              <X size={20} />
            </button>
          </div>
          <p className="small">
            {tr(
              'Записи объединяются по ID. Ошибка в пакете отменяет весь импорт. Завершения из истории не начисляют EXP.',
              'Жазбалар ID бойынша біріктіріледі. Пакеттегі қате бүкіл импортты тоқтатады. Тарихтағы орындалғандар EXP бермейді.',
              'Records merge by ID. A validation error prevents the whole import. Imported completions do not award EXP.',
            )}
          </p>
          <form onSubmit={upload}>
            <div className="upload-grid">
              <label>
                {tr(
                  'Профили сотрудников · JSON',
                  'Қызметкер профильдері · JSON',
                  'Employee profiles · JSON',
                )}
                <input type="file" name="employees" accept=".json,application/json" />
              </label>
              <label>
                {tr('История участия · CSV', 'Қатысу тарихы · CSV', 'Participation history · CSV')}
                <input type="file" name="history" accept=".csv,text/csv" />
              </label>
            </div>
            {importError && <ErrorMessage message={importError} />}{' '}
            {importDetails.length > 0 && (
              <div className="table-scroll">
                <table>
                  <thead>
                    <tr>
                      <th>{tr('Файл', 'Файл', 'File')}</th>
                      <th>{tr('Запись', 'Жазба', 'Record')}</th>
                      <th>{tr('Что исправить', 'Нені түзету керек', 'What to fix')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {importDetails.map((issue, index) => (
                      <tr key={index}>
                        <td>{issue.file}</td>
                        <td>{issue.record || '—'}</td>
                        <td>{issue.reason}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <button className="button primary" disabled={busy}>
              {busy
                ? tr('Проверяем и сохраняем…', 'Тексеріп сақтаудамыз…', 'Validating and saving…')
                : tr('Проверить и импортировать', 'Тексеру және импорттау', 'Validate and import')}
              <ArrowRight size={16} />
            </button>
          </form>
        </section>
      )}
      <div
        className="tabs"
        role="group"
        aria-label={tr('Разделы HR', 'HR бөлімдері', 'HR sections')}
      >
        <button
          className={tab === 'reviews' ? 'active' : ''}
          aria-pressed={tab === 'reviews'}
          onClick={() => setTab('reviews')}
        >
          <ClipboardCheck size={17} />
          {tr('Проверки', 'Тексерулер', 'Reviews')}
        </button>
        <button
          className={tab === 'employees' ? 'active' : ''}
          aria-pressed={tab === 'employees'}
          onClick={() => setTab('employees')}
        >
          <Users size={17} />
          {tr('Сотрудники', 'Қызметкерлер', 'Employees')}
          <span>{employees.length}</span>
        </button>
        <button
          className={tab === 'analytics' ? 'active' : ''}
          aria-pressed={tab === 'analytics'}
          onClick={() => setTab('analytics')}
        >
          <BarChart3 size={17} />
          {tr('Развитие команды', 'Команданың дамуы', 'Team growth')}
        </button>
      </div>
      {tab === 'reviews' && (
        <ReviewQueue employees={employees} session={session} onEmployee={onEmployee} />
      )}
      {tab === 'employees' && (
        <>
          <div className="filters">
            <label className="search">
              <Search size={17} />
              <input
                aria-label={tr('Поиск сотрудника', 'Қызметкерді іздеу', 'Search employees')}
                placeholder={tr('Имя, роль или ID…', 'Аты, рөлі немесе ID…', 'Name, role or ID…')}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
              />
            </label>
            <select
              aria-label={tr('Подразделение', 'Бөлім', 'Department')}
              value={department}
              onChange={(e) => setDepartment(e.target.value)}
            >
              <option value="">
                {tr('Все подразделения', 'Барлық бөлімдер', 'All departments')}
              </option>
              {[...new Set(employees.map((e) => e.department))].sort().map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
            <select
              aria-label={tr('Роль', 'Рөл', 'Role')}
              value={role}
              onChange={(e) => setRole(e.target.value)}
            >
              <option value="">{tr('Все роли', 'Барлық рөлдер', 'All roles')}</option>
              {[...new Set(employees.map((e) => e.role))].sort().map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
            <select
              aria-label={tr('Грейд', 'Деңгей', 'Grade')}
              value={grade}
              onChange={(e) => setGrade(e.target.value)}
            >
              <option value="">{tr('Все уровни', 'Барлық деңгейлер', 'All grades')}</option>
              {[...new Set(employees.map((e) => e.grade))].sort().map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
          </div>
          <section className="panel">
            <div className="employee-list table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>{tr('Сотрудник', 'Қызметкер', 'Employee')}</th>
                    <th>{tr('Роль', 'Рөл', 'Role')}</th>
                    <th>{tr('Грейд', 'Деңгей', 'Grade')}</th>
                    <th>{tr('Подразделение', 'Бөлім', 'Department')}</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {filtered.map((e) => (
                    <tr key={e.employee_id}>
                      <td>
                        <button className="employee-link" onClick={() => onEmployee(e.employee_id)}>
                          {e.full_name}
                          <small>{e.employee_id}</small>
                        </button>
                      </td>
                      <td>{e.role}</td>
                      <td>
                        <span className="tag subtle">{e.grade}</span>
                      </td>
                      <td>{e.department}</td>
                      <td>
                        <button
                          className="icon-button"
                          aria-label={`${tr('Открыть', 'Ашу', 'Open')} ${e.full_name}`}
                          onClick={() => onEmployee(e.employee_id)}
                        >
                          <ArrowRight size={17} />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {!filtered.length && (
                <Empty
                  title={tr('Никого не нашли', 'Ешкім табылмады', 'No matches')}
                  text={tr(
                    'Измените запрос или фильтры.',
                    'Сұрауды немесе сүзгілерді өзгертіңіз.',
                    'Try changing your search or filters.',
                  )}
                />
              )}
            </div>
          </section>
        </>
      )}
      {tab === 'analytics' && (
        <>
          {error && <ErrorMessage message={error} retry={() => setRevision((v) => v + 1)} />}{' '}
          {!overview && !error && <Loading />}
          {overview && (
            <>
              <div className="stats-grid">
                <Stat
                  label={tr('Сотрудников', 'Қызметкерлер', 'Employees')}
                  value={overview.employee_count}
                  note={tr(
                    'в пространстве развития',
                    'даму кеңістігінде',
                    'in the development workspace',
                  )}
                />
                <Stat
                  label={tr('Средний прогресс', 'Орташа прогресс', 'Average readiness')}
                  value={`${overview.average_progress}%`}
                  note={tr(
                    'по требованиям личных целей',
                    'жеке мақсаттар талаптары бойынша',
                    'against individual goal requirements',
                  )}
                />
                <Stat
                  label={tr('Нужен следующий шаг', 'Келесі қадам қажет', 'Need a next step')}
                  value={overview.without_next_step.length}
                  note={tr(
                    'повод помочь с выбором',
                    'таңдауға көмектесу мүмкіндігі',
                    'an opportunity to help',
                  )}
                />
                <Stat
                  label={tr('Модулей в каталоге', 'Каталогтағы модульдер', 'Catalog modules')}
                  value={overview.event_count}
                  note={`${overview.skill_count} ${tr('навыков', 'дағды', 'skills')}`}
                />
              </div>
              <div className="hr-grid section">
                <section className="panel hr-panel">
                  <div className="section-heading">
                    <div>
                      <span className="eyebrow">
                        {tr('ГДЕ НУЖНА ПОДДЕРЖКА', 'ҚАЙ ЖЕРДЕ ҚОЛДАУ КЕРЕК', 'WHERE SUPPORT HELPS')}
                      </span>
                      <h2>{tr('Дефициты навыков', 'Дағды тапшылығы', 'Skill gaps')}</h2>
                    </div>
                  </div>
                  <p className="small">
                    {tr(
                      'Сотрудники с разрывом / сотрудники, которым навык нужен для цели.',
                      'Дағды жетіспейтін қызметкерлер / мақсатқа осы дағды қажет қызметкерлер.',
                      'Employees with a gap / employees who need the skill for their goal.',
                    )}
                  </p>
                  <div className="gap-bars">
                    {overview.gaps.slice(0, 10).map((g, index) => (
                      <div className="gap-bar" key={g.skill_id}>
                        <div>
                          <span>{g.name}</span>
                          <strong>
                            {g.count}
                            <small> / {g.required_by}</small>
                          </strong>
                        </div>
                        <div className="bar-track">
                          <span
                            className={index < 3 ? 'emphasis' : ''}
                            style={{
                              width: `${g.required_by ? (g.count / g.required_by) * 100 : 0}%`,
                            }}
                          />
                        </div>
                        <small>
                          {g.critical_count}{' '}
                          {tr('критических разрывов', 'маңызды тапшылық', 'critical gaps')}
                        </small>
                      </div>
                    ))}
                  </div>
                </section>
                <section className="panel hr-panel">
                  <div className="section-heading">
                    <div>
                      <span className="eyebrow">
                        {tr('ПОМОЧЬ С ВЫБОРОМ', 'ТАҢДАУҒА КӨМЕКТЕСУ', 'HELP FIND A WAY FORWARD')}
                      </span>
                      <h2>{tr('Без следующего шага', 'Келесі қадамсыз', 'No next step yet')}</h2>
                    </div>
                    <span className="count-bubble">{overview.without_next_step.length}</span>
                  </div>
                  <div className="no-step-list">
                    {overview.without_next_step.length ? (
                      overview.without_next_step.map((e) => (
                        <button
                          className="no-step-item"
                          key={e.employee_id}
                          onClick={() => onEmployee(e.employee_id)}
                        >
                          <span className="mini-avatar">{e.full_name[0]}</span>
                          <span>
                            <strong>{e.full_name}</strong>
                            <small>{e.department}</small>
                            <p>{e.reason}</p>
                          </span>
                          <ArrowRight size={16} />
                        </button>
                      ))
                    ) : (
                      <Empty
                        title={tr(
                          'У всех есть следующий шаг',
                          'Барлығында келесі қадам бар',
                          'Everyone has a next step',
                        )}
                        text={tr(
                          'Подходящие модули найдены для всей команды.',
                          'Бүкіл команда үшін қолайлы модульдер табылды.',
                          'Suitable modules were found for the whole team.',
                        )}
                      />
                    )}
                  </div>
                </section>
              </div>
              <section className="panel section">
                <div className="panel-heading">
                  <div>
                    <h2>
                      {tr('Участие в активностях', 'Іс-шараларға қатысу', 'Activity participation')}
                    </h2>
                    <p className="small">
                      {tr('Данные на', 'Деректер күні', 'Data as of')}{' '}
                      {formatDate(overview.as_of_date, locale)}
                    </p>
                  </div>
                </div>
                <div className="table-scroll">
                  <table>
                    <thead>
                      <tr>
                        <th>{tr('Активность', 'Іс-шара', 'Activity')}</th>
                        <th>{tr('Участий', 'Қатысулар', 'Participations')}</th>
                        <th>{tr('Завершено', 'Аяқталған', 'Completed')}</th>
                        <th>{tr('Статусы', 'Күйлер', 'Statuses')}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {overview.participation.map((p) => (
                        <tr key={p.event_id}>
                          <td>
                            <strong>{p.title}</strong>
                            {p.mandatory && (
                              <small className="muted block">
                                {tr(
                                  'Обязательная · без EXP',
                                  'Міндетті · EXP жоқ',
                                  'Mandatory · no EXP',
                                )}
                              </small>
                            )}
                          </td>
                          <td>{p.total}</td>
                          <td>
                            {p.completed} / {p.total}
                          </td>
                          <td>
                            <div className="status-chips">
                              {Object.entries(p.statuses).map(([state, count]) => (
                                <span key={state} className="quiet-tag">
                                  {labels.status(state)}: {count}
                                </span>
                              ))}
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </section>
            </>
          )}
        </>
      )}
    </>
  );
}
