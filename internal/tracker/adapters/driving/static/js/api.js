export class API {
    constructor(projectId) {
        this.projectId = projectId;
    }

    async fetchGoals() {
        const response = await fetch(`/api/goals?projectId=${encodeURIComponent(this.projectId)}`);
        if (!response.ok) {
            const text = await response.text();
            throw new Error(text || 'Failed to fetch goals');
        }
        return await response.json();
    }

    async moveGoal(childId, newParentId) {
        const response = await fetch('/api/goals/move', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                projectId: this.projectId,
                childId: childId,
                newParentId: newParentId
            })
        });
        if (!response.ok) {
            const text = await response.text();
            throw new Error(text || 'Failed to move goal');
        }
        return await response.json();
    }
}
