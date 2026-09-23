import { useEffect, useState, type FormEvent } from 'react';
import { ArrowRight, BarChart3, CheckCircle2, FileUp, Search, Users, X } from 'lucide-react';
import { api } from '../api/client';
import type { EmployeeSummary, Overview } from '../api/types';
import { dateLabel, Empty, ErrorMessage, Loading, Stat, statusLabels } from '../components/Shared';

export default function HR({
  employees,
  onEmployee,
  onImport,
}: {
  employees: EmployeeSummary[];
  onEmployee: (id: string) => void;
  onImport: () => void;
}) {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [error, setError] = useState('');
  const [query, setQuery] = useState('');
  const [showImport, setShowImport] = useState(false);
  const [busy, setBusy] = useState(false);
  const [importError, setImportError] = useState('');
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
  }, [revision]);
  async function upload(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = new FormData(form);
    for (const key of ['employees', 'history']) {
      const file = data.get(key);
      if (file instanceof File && !file.size) data.delete(key);
    }
    if (![...data.keys()].length) {
      setImportError('Выберите хотя бы один файл.');
      return;
    }
    setBusy(true);
    setImportError('');
    setNotice('');
    try {
      const result = await api.import(data);
      setNotice(
        `Импорт завершён: сотрудников добавлено ${result.employees_added}, обновлено ${result.employees_updated}; записей истории добавлено ${result.history_added}, обновлено ${result.history_updated}.`,
      );
      form.reset();
      setShowImport(false);
      setRevision((v) => v + 1);
      onImport();
    } catch (err) {
      setImportError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  if (error) return <ErrorMessage message={error} retry={() => setRevision((v) => v + 1)} />;
  if (!overview) return <Loading />;
  const filtered = employees.filter((e) =>
    `${e.full_name} ${e.employee_id} ${e.role} ${e.department}`
      .toLowerCase()
      .includes(query.toLowerCase()),
  );
  return (
    <>
      <header className="page-heading">
        <div>
          <div className="eyebrow">HR-ПРОСТРАНСТВО</div>
          <h1>
            Развитие в фокусе<span className="heading-dot">.</span>
          </h1>
          <p>Замечайте дефициты навыков и помогайте команде двигаться дальше.</p>
        </div>
        <button className="button primary" onClick={() => setShowImport((v) => !v)}>
          <FileUp size={17} />
          Загрузить данные
        </button>
      </header>
      {notice && (
        <div className="success" role="status">
          <CheckCircle2 size={18} />
          {notice}
        </div>
      )}
      {showImport && (
        <section className="panel import-panel">
          <div className="section-heading">
            <h2>Проверочные данные</h2>
            <button
              className="icon-button"
              aria-label="Закрыть импорт"
              onClick={() => setShowImport(false)}
            >
              <X size={20} />
            </button>
          </div>
          <p className="muted">
            Профили и история объединяются по ID. Весь пакет проверяется до сохранения. Дата среза:{' '}
            {overview.as_of_date}.
          </p>
          <form onSubmit={upload}>
            <div className="upload-grid">
              <label>
                Профили сотрудников (.json)
                <input type="file" name="employees" accept=".json,application/json" />
              </label>
              <label>
                История участия (.csv)
                <input type="file" name="history" accept=".csv,text/csv" />
              </label>
            </div>
            {importError && <ErrorMessage message={importError} />}
            <button className="button primary" disabled={busy}>
              {busy ? 'Проверяем и сохраняем…' : 'Проверить и импортировать'}
              <ArrowRight size={16} />
            </button>
          </form>
        </section>
      )}
      <div className="stats-grid">
        <Stat label="Сотрудников" value={overview.employee_count} note="в пространстве развития" />
        <Stat
          label="Средний прогресс"
          value={`${overview.average_progress}%`}
          note="относительно карьерной цели"
        />
        <Stat
          label="Нужен следующий шаг"
          value={overview.without_next_step.length}
          note="нет подходящей активности"
        />
        <Stat
          label="Активностей в каталоге"
          value={overview.event_count}
          note={`${overview.skill_count} навыков · ${overview.history_count} записей истории`}
        />
      </div>
      <div className="hr-grid section">
        <section className="panel hr-panel">
          <div className="section-heading">
            <div>
              <span className="eyebrow">ГДЕ НУЖНА ПОДДЕРЖКА</span>
              <h2>
                <BarChart3 size={21} />
                Дефициты компетенций
              </h2>
            </div>
          </div>
          <p className="muted small">
            Число сотрудников с разрывом / число сотрудников, которым навык нужен для цели.
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
                    style={{ width: `${(g.count / g.required_by) * 100}%` }}
                  />
                </div>
                <small>
                  {g.critical_count > 0
                    ? `${g.critical_count} — критический разрыв`
                    : 'Навык для развития'}
                </small>
              </div>
            ))}
          </div>
        </section>
        <section className="panel hr-panel">
          <div className="section-heading">
            <div>
              <span className="eyebrow">ПОМОЧЬ С ВЫБОРОМ</span>
              <h2>
                <Users size={21} />
                Без следующего шага
              </h2>
            </div>
            <span className="count-bubble">{overview.without_next_step.length}</span>
          </div>
          <p className="muted small">
            Цель уже достигнута или в каталоге пока нет подходящего шага.
          </p>
          <div className="no-step-list">
            {overview.without_next_step.length ? (
              overview.without_next_step.map((e) => (
                <button
                  className="no-step-item"
                  key={e.employee_id}
                  onClick={() => onEmployee(e.employee_id)}
                >
                  <span className="mini-avatar">{e.full_name.slice(0, 1)}</span>
                  <span>
                    <strong>{e.full_name}</strong>
                    <small>{e.department}</small>
                    <p>{e.reason}</p>
                  </span>
                  <ArrowRight size={17} />
                </button>
              ))
            ) : (
              <Empty
                title="У всех есть следующий шаг"
                text="Доступные активности найдены для всех сотрудников."
              />
            )}
          </div>
        </section>
      </div>
      <section className="panel section">
        <div className="panel-heading">
          <div>
            <h2>Команда</h2>
            <p className="muted small">
              Откройте профиль, чтобы увидеть траекторию и рекомендации.
            </p>
          </div>
          <label className="search">
            <Search size={17} />
            <input
              aria-label="Поиск сотрудника"
              placeholder="Имя, роль или ID…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </label>
        </div>
        <div className="employee-list table-scroll">
          <table>
            <thead>
              <tr>
                <th>Сотрудник</th>
                <th>Роль</th>
                <th>Грейд</th>
                <th>Подразделение</th>
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
                      aria-label={`Открыть ${e.full_name}`}
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
            <Empty title="Никого не нашли" text="Попробуйте другое имя или ID сотрудника." />
          )}
        </div>
      </section>
      <section className="panel section">
        <div className="panel-heading">
          <div>
            <h2>Участие в активностях</h2>
            <p className="muted small">
              Все статусы участия на {dateLabel(overview.as_of_date)}. Повторные встречи считаются
              отдельными участиями.
            </p>
          </div>
        </div>
        <div className="table-scroll participation-table">
          <table>
            <thead>
              <tr>
                <th>Активность</th>
                <th>Участий</th>
                <th>Завершено</th>
                <th>Статусы</th>
              </tr>
            </thead>
            <tbody>
              {overview.participation.map((p) => (
                <tr key={p.event_id}>
                  <td>
                    <strong>{p.title}</strong>
                    {p.mandatory && <small className="muted block">Обязательная</small>}
                  </td>
                  <td>{p.total}</td>
                  <td>
                    <span className="skill-done">
                      {p.completed} / {p.total}
                    </span>
                  </td>
                  <td>
                    <div className="status-chips">
                      {Object.entries(p.statuses).map(([status, count]) => (
                        <span key={status} className="quiet-tag">
                          {statusLabels[status]}: {count}
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
  );
}
