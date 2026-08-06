import { API } from "./api.js";
import { Drawer } from "./components/drawer.js";
import { MindMap } from "./mindmap/mindmap.js";

class App {
	constructor() {
		this.projectId =
			new URLSearchParams(window.location.search).get("projectId") || "";
		this.api = new API(this.projectId);
		this.drawer = new Drawer();
		this.drawer.setOnUpdate(async (goalId, options) => {
			await this.api.updateGoal(goalId, options);
			// Fetch only the single updated goal to patch the card in place
			// without recomputing the full layout (which shifts node positions).
			const goals = await this.api.fetchGoals();
			const updatedGoal = goals.find((g) => g.id === goalId);
			if (updatedGoal) {
				this.mindmap.patchGoal(updatedGoal);
				if (this.drawer.currentGoal && this.drawer.currentGoal.id === goalId) {
					this.drawer.show(updatedGoal);
				}
			}
		});

		// Initialize mindmap component, passing API and drawer for interactions
		this.mindmap = new MindMap("#mindmap-container", this.api, this.drawer);

		this.init();
	}

	async init() {
		try {
			const [goals, members] = await Promise.all([
				this.api.fetchGoals(),
				this.api.fetchMembers(),
			]);
			this.drawer.setMembers(members || []);
			if (!goals || goals.length === 0) {
				alert("No goals found for this project.");
				return;
			}
			this.mindmap.render(goals, true);
		} catch (err) {
			console.error("Failed to initialize Sattchel Visualizer:", err);
			alert("Failed to load goals: " + err.message);
		}
	}
}

// Start the app when DOM is ready
document.addEventListener("DOMContentLoaded", () => {
	window.app = new App();
});
