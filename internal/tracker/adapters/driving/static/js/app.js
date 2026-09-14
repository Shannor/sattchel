import { API } from "./api.js";
import { Drawer } from "./components/drawer.js";
import { MindMap } from "./mindmap/mindmap.js";

class App {
	constructor() {
		this.projectId =
			new URLSearchParams(window.location.search).get("projectId") || "";
		this.api = new API(this.projectId);
		this.drawer = new Drawer();

		this.projects = [];
		this.allGoals = [];
		this.members = [];
		this.filterState = {
			hideCompleted: false,
			statusFilter: "all",
			memberFilter: "all",
			searchQuery: "",
		};

		this.drawer.setOnUpdate(async (goalId, options) => {
			await this.api.updateGoal(goalId, options);
			const goals = await this.api.fetchGoals();
			this.allGoals = goals || [];

			const updatedGoal = this.allGoals.find((g) => g.id === goalId);
			if (updatedGoal) {
				if (
					this.drawer.currentGoal &&
					this.drawer.currentGoal.id === goalId
				) {
					this.drawer.show(updatedGoal);
				}

				const isFilteringActive =
					this.filterState.hideCompleted ||
					this.filterState.statusFilter !== "all" ||
					this.filterState.memberFilter !== "all" ||
					this.filterState.searchQuery !== "";

				if (isFilteringActive) {
					this.applyFilters(false);
				} else {
					this.mindmap.patchGoal(updatedGoal);
				}
			} else {
				this.applyFilters(false);
			}
		});

		// Initialize mindmap component
		this.mindmap = new MindMap("#mindmap-container", this.api, this.drawer);
		this.mindmap.setOnGoalsChanged((goals) => {
			this.allGoals = goals;
			this.applyFilters(false);
		});

		this.initFilterControls();
		this.init();
	}

	initFilterControls() {
		const selectProject = document.getElementById("select-project");
		const btnHideCompleted = document.getElementById("btn-hide-completed");
		const selectStatus = document.getElementById("filter-status");
		const selectMember = document.getElementById("filter-member");
		const inputSearch = document.getElementById("filter-search");

		if (selectProject) {
			selectProject.addEventListener("change", async (e) => {
				const newPid = e.target.value;
				if (!newPid || newPid === this.projectId) return;

				this.projectId = newPid;
				this.api.projectId = newPid;

				try {
					const url = new URL(window.location.href);
					url.searchParams.set("projectId", newPid);
					window.history.pushState({}, "", url);
				} catch {
					// Ignore URL parsing errors
				}

				this.mindmap.rootChildSides = {};

				try {
					const goals = await this.api.fetchGoals();
					this.allGoals = goals || [];
					if (this.drawer && this.drawer.close) {
						this.drawer.close();
					}
					if (this.allGoals.length === 0) {
						this.mindmap.render([], true);
						const filterCount = document.getElementById("filter-count");
						if (filterCount) filterCount.textContent = "0 goals";
						return;
					}
					this.applyFilters(true);
				} catch (err) {
					console.error("Failed to load goals for project:", err);
					alert("Failed to load goals for selected project: " + err.message);
				}
			});
		}

		if (btnHideCompleted) {
			btnHideCompleted.addEventListener("click", () => {
				this.filterState.hideCompleted = !this.filterState.hideCompleted;
				btnHideCompleted.classList.toggle("active", this.filterState.hideCompleted);

				if (selectStatus) {
					if (this.filterState.hideCompleted && selectStatus.value === "all") {
						selectStatus.value = "hide-completed";
						this.filterState.statusFilter = "hide-completed";
					} else if (
						!this.filterState.hideCompleted &&
						selectStatus.value === "hide-completed"
					) {
						selectStatus.value = "all";
						this.filterState.statusFilter = "all";
					}
				}

				this.applyFilters(false);
			});
		}

		if (selectStatus) {
			selectStatus.addEventListener("change", (e) => {
				const val = e.target.value;
				this.filterState.statusFilter = val;
				if (val === "hide-completed") {
					this.filterState.hideCompleted = true;
				} else {
					this.filterState.hideCompleted = false;
				}

				if (btnHideCompleted) {
					btnHideCompleted.classList.toggle(
						"active",
						this.filterState.hideCompleted,
					);
				}

				this.applyFilters(false);
			});
		}

		if (selectMember) {
			selectMember.addEventListener("change", (e) => {
				this.filterState.memberFilter = e.target.value;
				this.applyFilters(false);
			});
		}

		if (inputSearch) {
			inputSearch.addEventListener("input", (e) => {
				this.filterState.searchQuery = e.target.value.trim().toLowerCase();
				this.applyFilters(false);
			});
		}
	}

	populateProjectFilter(projects) {
		const selectProject = document.getElementById("select-project");
		if (!selectProject) return;

		selectProject.textContent = "";

		(projects || []).forEach((p) => {
			const opt = document.createElement("option");
			opt.value = p.id;
			opt.textContent = p.label || p.id;
			selectProject.appendChild(opt);
		});

		if (this.projectId) {
			selectProject.value = this.projectId;
		} else if (projects && projects.length > 0) {
			selectProject.value = projects[0].id;
			this.projectId = projects[0].id;
			this.api.projectId = projects[0].id;
		}
	}

	populateMemberFilter(members) {
		const selectMember = document.getElementById("filter-member");
		if (!selectMember) return;

		selectMember.textContent = "";

		const optAll = document.createElement("option");
		optAll.value = "all";
		optAll.textContent = "All Members";
		selectMember.appendChild(optAll);

		const optUnassigned = document.createElement("option");
		optUnassigned.value = "unassigned";
		optUnassigned.textContent = "Unassigned";
		selectMember.appendChild(optUnassigned);

		(members || []).forEach((m) => {
			const opt = document.createElement("option");
			opt.value = m.id;
			opt.textContent = m.name;
			selectMember.appendChild(opt);
		});
	}

	filterGoals(goals) {
		const isRoot = (g) => !g.parent || !g.parent.targetId;

		// 1. Identify visible goal IDs
		const visibleIds = new Set();
		goals.forEach((g) => {
			if (isRoot(g)) {
				visibleIds.add(g.id);
				return;
			}

			const statusNorm = (g.status || "draft")
				.toLowerCase()
				.trim()
				.replace(" ", "-");
			const isCompleted = statusNorm === "completed";

			if (this.filterState.hideCompleted && isCompleted) {
				return;
			}

			if (
				this.filterState.statusFilter !== "all" &&
				this.filterState.statusFilter !== "hide-completed" &&
				statusNorm !== this.filterState.statusFilter
			) {
				return;
			}

			if (this.filterState.memberFilter !== "all") {
				const memberId = g.member ? g.member.id : "";
				if (this.filterState.memberFilter === "unassigned") {
					if (memberId !== "") return;
				} else if (memberId !== this.filterState.memberFilter) {
					return;
				}
			}

			if (this.filterState.searchQuery) {
				const q = this.filterState.searchQuery;
				const nameMatch = (g.name || "").toLowerCase().includes(q);
				const descMatch = (g.description || "").toLowerCase().includes(q);
				if (!nameMatch && !descMatch) {
					return;
				}
			}

			visibleIds.add(g.id);
		});

		// 2. Map goals by ID
		const goalMap = {};
		goals.forEach((g) => {
			goalMap[g.id] = g;
		});

		// 3. Re-link goals to nearest visible ancestor
		return goals
			.filter((g) => visibleIds.has(g.id))
			.map((g) => {
				if (isRoot(g)) return g;

				let parent = g.parent;
				if (parent && parent.targetId) {
					let targetId = parent.targetId;
					while (targetId && !visibleIds.has(targetId)) {
						const parentGoal = goalMap[targetId];
						if (parentGoal && parentGoal.parent && parentGoal.parent.targetId) {
							targetId = parentGoal.parent.targetId;
						} else {
							targetId = null;
						}
					}
					if (targetId) {
						parent = { ...parent, targetId: targetId };
					} else {
						parent = null;
					}
				}
				return { ...g, parent };
			});
	}

	applyFilters(isInitial = false) {
		if (!this.allGoals) return;

		const prevScale = this.mindmap.zoomPan ? this.mindmap.zoomPan.scale : 1;
		const prevTx = this.mindmap.zoomPan ? this.mindmap.zoomPan.translateX : 0;
		const prevTy = this.mindmap.zoomPan ? this.mindmap.zoomPan.translateY : 0;

		const filtered = this.filterGoals(this.allGoals);
		this.mindmap.render(filtered, isInitial);

		if (!isInitial && this.mindmap.zoomPan) {
			this.mindmap.zoomPan.scale = prevScale;
			this.mindmap.zoomPan.translateX = prevTx;
			this.mindmap.zoomPan.translateY = prevTy;
			this.mindmap.zoomPan.updateTransform();
		}

		const filterCount = document.getElementById("filter-count");
		if (filterCount) {
			const isFiltering =
				this.filterState.hideCompleted ||
				this.filterState.statusFilter !== "all" ||
				this.filterState.memberFilter !== "all" ||
				this.filterState.searchQuery !== "";
			if (isFiltering) {
				filterCount.textContent = `Showing ${filtered.length} of ${this.allGoals.length} goals`;
			} else {
				filterCount.textContent = `${this.allGoals.length} goals`;
			}
		}
	}

	async init() {
		try {
			const [projects, goals, members] = await Promise.all([
				this.api.fetchProjects(),
				this.api.fetchGoals(),
				this.api.fetchMembers(),
			]);
			this.projects = projects || [];
			this.allGoals = goals || [];
			this.members = members || [];

			this.populateProjectFilter(this.projects);
			this.drawer.setMembers(this.members);
			this.populateMemberFilter(this.members);

			if (!goals || goals.length === 0) {
				const filterCount = document.getElementById("filter-count");
				if (filterCount) filterCount.textContent = "0 goals";
				return;
			}

			this.applyFilters(true);
		} catch (err) {
			console.error("Failed to initialize Sattchel Visualizer:", err);
			alert("Failed to load visualizer data: " + err.message);
		}
	}
}

// Start the app when DOM is ready
document.addEventListener("DOMContentLoaded", () => {
	window.app = new App();
});
