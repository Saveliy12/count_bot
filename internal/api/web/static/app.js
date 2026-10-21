const API = "/api/v1";
const PAGE_SIZE = 50;

const moneyWhole = new Intl.NumberFormat("ru-RU", {
  style: "currency",
  currency: "RUB",
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

const moneyExact = new Intl.NumberFormat("ru-RU", {
  style: "currency",
  currency: "RUB",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

const fieldMessages = {
  amount: "Сумма должна быть положительной и не больше двух знаков после запятой.",
  category: "Неизвестная категория.",
  spent_at: "Проверьте дату траты.",
  comment: "Комментарий слишком длинный.",
  user_id: "Выберите участника.",
  from: "Проверьте дату начала периода.",
  to: "Проверьте дату конца периода.",
  period: "Дата начала периода позже даты конца.",
};

const state = {
  users: [],
  categories: [],
  items: [],
  total: "0.00",
  count: 0,
  hasMore: false,
  offset: 0,
  loading: false,
  requestId: 0,
  filters: {
    userId: "",
    category: "",
    preset: "month",
    from: "",
    to: "",
  },
};

const els = {
  total: document.querySelector("#total"),
  count: document.querySelector("#count"),
  periodLabel: document.querySelector("#period-label"),
  member: document.querySelector("#member-filter"),
  period: document.querySelector("#period-filter"),
  category: document.querySelector("#category-filter"),
  customDates: document.querySelector("#custom-dates"),
  from: document.querySelector("#from-filter"),
  to: document.querySelector("#to-filter"),
  reset: document.querySelector("#reset-filters"),
  add: document.querySelector("#add-expense"),
  status: document.querySelector("#status"),
  tableWrap: document.querySelector("#table-wrap"),
  body: document.querySelector("#expenses-body"),
  empty: document.querySelector("#empty"),
  emptyTitle: document.querySelector("#empty-title"),
  listHint: document.querySelector("#list-hint"),
  loadMore: document.querySelector("#load-more"),
  retry: document.querySelector("#retry"),
  listPanel: document.querySelector("#list-panel"),
};

async function api(path, options = {}) {
  const response = await fetch(path, options);
  if (!response.ok) {
    let body = null;
    try {
      body = await response.json();
    } catch {
      // Some errors may not have a JSON body.
    }

    const error = new Error(body?.error?.message || `HTTP ${response.status}`);
    error.status = response.status;
    error.body = body;
    throw error;
  }

  if (response.status === 204) {
    return null;
  }

  return response.json();
}

function localDate(date = new Date()) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function shiftDays(date, days) {
  const copy = new Date(date);
  copy.setDate(copy.getDate() + days);
  return copy;
}

function firstOfMonth(date) {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}

function formatDate(value) {
  if (!value) {
    return "—";
  }

  const [year, month, day] = value.split("-").map(Number);
  return new Date(year, month - 1, day).toLocaleDateString("ru-RU");
}

function formatMoney(value) {
  const number = Number(value);
  if (!Number.isFinite(number)) {
    return "— ₽";
  }

  return Math.round(number * 100) % 100 === 0
    ? moneyWhole.format(number)
    : moneyExact.format(number);
}

function plural(count, one, few, many) {
  const mod10 = count % 10;
  const mod100 = count % 100;
  if (mod10 === 1 && mod100 !== 11) {
    return one;
  }
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) {
    return few;
  }
  return many;
}

function countLabel(count) {
  return `${count} ${plural(count, "трата", "траты", "трат")}`;
}

function applyPreset(preset, resetDates = false) {
  const today = new Date();
  state.filters.preset = preset;

  switch (preset) {
    case "today":
      state.filters.from = localDate(today);
      state.filters.to = localDate(today);
      break;
    case "week":
      state.filters.from = localDate(shiftDays(today, -6));
      state.filters.to = localDate(today);
      break;
    case "month":
      state.filters.from = localDate(firstOfMonth(today));
      state.filters.to = localDate(today);
      break;
    case "all":
      state.filters.from = "";
      state.filters.to = "";
      break;
    case "custom":
      if (resetDates || !state.filters.from || !state.filters.to) {
        state.filters.from = localDate(firstOfMonth(today));
        state.filters.to = localDate(today);
      }
      break;
    default:
      state.filters.preset = "month";
      state.filters.from = localDate(firstOfMonth(today));
      state.filters.to = localDate(today);
  }

  els.period.value = state.filters.preset;
  els.from.value = state.filters.from;
  els.to.value = state.filters.to;
  els.customDates.hidden = state.filters.preset !== "custom";
}

function periodText() {
  switch (state.filters.preset) {
    case "today":
      return "сегодня";
    case "week":
      return "последние 7 дней";
    case "month":
      return "этот месяц";
    case "all":
      return "всё время";
    case "custom":
      if (state.filters.from && state.filters.to) {
        return `${formatDate(state.filters.from)} — ${formatDate(state.filters.to)}`;
      }
      if (state.filters.from) {
        return `с ${formatDate(state.filters.from)}`;
      }
      if (state.filters.to) {
        return `до ${formatDate(state.filters.to)}`;
      }
      return "выбранный период";
    default:
      return "";
  }
}

function setStatus(message, isError = false) {
  els.status.textContent = message;
  els.status.classList.toggle("status--error", isError);
}

function formatError(error) {
  const field = error?.body?.error?.field;
  if (field && fieldMessages[field]) {
    return fieldMessages[field];
  }

  if (error?.status >= 500) {
    return "Сервер временно недоступен.";
  }

  return error?.body?.error?.message || error?.message || "Неизвестная ошибка.";
}

function fillSelect(select, placeholder, items, valueOf, labelOf) {
  const current = select.value;
  select.replaceChildren(new Option(placeholder, ""));
  for (const item of items) {
    select.append(new Option(labelOf(item), String(valueOf(item))));
  }

  if ([...select.options].some((option) => option.value === current)) {
    select.value = current;
  }
}

function memberName(userId) {
  const user = state.users.find((item) => String(item.telegram_id) === String(userId));
  return user?.display_name || `Пользователь ${userId}`;
}

function cell(label, value, className = "") {
  const element = document.createElement("td");
  element.dataset.label = label;
  if (className) {
    element.className = className;
  }
  element.textContent = value;
  return element;
}

function memberCell(userId) {
  const element = document.createElement("td");
  element.dataset.label = "Кто";

  const member = document.createElement("span");
  member.className = "member";
  member.textContent = memberName(userId);
  element.append(member);

  return element;
}

function categoryCell(value) {
  const category = state.categories.find((item) => item.value === value);
  const element = document.createElement("td");
  element.dataset.label = "Категория";

  const wrapper = document.createElement("span");
  wrapper.className = "category";

  const dot = document.createElement("i");
  dot.className = "dot";
  dot.dataset.category = value;

  wrapper.append(dot, document.createTextNode(category?.label || value));
  element.append(wrapper);

  return element;
}

function amountCell(value) {
  const element = document.createElement("td");
  element.className = "amount";
  element.dataset.label = "Сумма";
  element.textContent = formatMoney(value);
  return element;
}

function renderRows() {
  els.body.replaceChildren();

  for (const expense of state.items) {
    const row = document.createElement("tr");
    row.append(
      cell("Дата", formatDate(expense.spent_at)),
      memberCell(expense.user_id),
      categoryCell(expense.category),
      cell("Комментарий", expense.comment || "—", "comment"),
      amountCell(expense.amount),
    );
    els.body.append(row);
  }
}

function renderSummary() {
  els.total.textContent = formatMoney(state.total);
  els.count.textContent = countLabel(state.count);
  els.periodLabel.textContent = periodText();
}

function render() {
  const hasItems = state.items.length > 0;

  if (state.loading && !hasItems) {
    els.total.textContent = "— ₽";
    els.count.textContent = "Загрузка…";
    els.periodLabel.textContent = periodText();
  } else {
    renderSummary();
  }

  renderRows();
  els.tableWrap.hidden = !hasItems;
  els.empty.hidden = hasItems || state.loading;
  els.emptyTitle.textContent = `За ${periodText()} трат нет.`;
  els.loadMore.hidden = !state.hasMore;
  els.listHint.textContent = state.count > 0
    ? `Показано ${state.items.length} из ${state.count}`
    : "";
  els.listPanel.setAttribute("aria-busy", state.loading ? "true" : "false");
}

function renderError(error) {
  setStatus(`Не удалось загрузить траты. ${formatError(error)}`, true);
  els.retry.hidden = false;
  els.loadMore.hidden = true;
  els.tableWrap.hidden = state.items.length === 0;
  els.empty.hidden = true;
  els.listPanel.setAttribute("aria-busy", "false");
}

async function loadExpenses({ append = false } = {}) {
  const requestId = ++state.requestId;
  state.loading = true;
  els.loadMore.disabled = true;
  els.retry.hidden = true;
  setStatus(append ? "Загружаем ещё…" : "Загрузка…");

  if (!append) {
    state.items = [];
    state.offset = 0;
    render();
  }

  const params = new URLSearchParams({
    limit: String(PAGE_SIZE),
    offset: String(state.offset),
  });

  if (state.filters.userId) {
    params.set("user_id", state.filters.userId);
  }
  if (state.filters.category) {
    params.set("category", state.filters.category);
  }
  if (state.filters.from) {
    params.set("from", state.filters.from);
  }
  if (state.filters.to) {
    params.set("to", state.filters.to);
  }

  try {
    const data = await api(`${API}/expenses?${params}`);
    if (requestId !== state.requestId) {
      return;
    }

    state.items = append ? state.items.concat(data.items) : data.items;
    state.total = data.total;
    state.count = data.count;
    state.hasMore = data.has_more;
    state.offset = state.items.length;
    state.loading = false;
    setStatus("");
    render();
  } catch (error) {
    if (requestId !== state.requestId) {
      return;
    }
    renderError(error);
  } finally {
    if (requestId === state.requestId) {
      state.loading = false;
      els.loadMore.disabled = false;
      els.listPanel.setAttribute("aria-busy", "false");
    }
  }
}

function resetAndLoad() {
  state.offset = 0;
  state.items = [];
  loadExpenses();
}

async function init() {
  setStatus("Загрузка…");
  els.retry.hidden = true;
  els.status.classList.remove("status--error");
  applyPreset("month", true);

  try {
    const [users, categories] = await Promise.all([
      api(`${API}/users`),
      api(`${API}/categories`),
    ]);

    state.users = users.items || [];
    state.categories = categories.items || [];

    fillSelect(
      els.member,
      "Вся семья",
      state.users,
      (user) => user.telegram_id,
      (user) => user.display_name || `Пользователь ${user.telegram_id}`,
    );
    fillSelect(
      els.category,
      "Все категории",
      state.categories,
      (category) => category.value,
      (category) => category.label,
    );

    els.member.value = state.filters.userId;
    els.category.value = state.filters.category;
    await loadExpenses();
  } catch (error) {
    els.total.textContent = "— ₽";
    els.count.textContent = "Нет данных";
    renderError(error);
  }
}

els.member.addEventListener("change", () => {
  state.filters.userId = els.member.value;
  resetAndLoad();
});

els.category.addEventListener("change", () => {
  state.filters.category = els.category.value;
  resetAndLoad();
});

els.period.addEventListener("change", () => {
  applyPreset(els.period.value, true);
  resetAndLoad();
});

els.from.addEventListener("change", () => {
  state.filters.preset = "custom";
  els.period.value = "custom";
  els.customDates.hidden = false;
  state.filters.from = els.from.value;
  resetAndLoad();
});

els.to.addEventListener("change", () => {
  state.filters.preset = "custom";
  els.period.value = "custom";
  els.customDates.hidden = false;
  state.filters.to = els.to.value;
  resetAndLoad();
});

els.reset.addEventListener("click", () => {
  state.filters.userId = "";
  state.filters.category = "";
  els.member.value = "";
  els.category.value = "";
  applyPreset("month", true);
  resetAndLoad();
});

els.loadMore.addEventListener("click", () => {
  loadExpenses({ append: true });
});

els.retry.addEventListener("click", () => {
  init();
});

init();
